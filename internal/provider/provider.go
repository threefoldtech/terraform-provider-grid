// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	"github.com/threefoldtech/rmb-sdk-go"
	"github.com/threefoldtech/rmb-sdk-go/direct"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

var (
	SUBSTRATE_URL = map[string]string{
		"dev":  "wss://tfchain.dev.grid.tf/ws",
		"test": "wss://tfchain.test.grid.tf/ws",
		"qa":   "wss://tfchain.qa.grid.tf/ws",
		"main": "wss://tfchain.grid.tf/ws",
	}
	RMB_PROXY_URL = map[string]string{
		"dev":  "https://gridproxy.dev.grid.tf/",
		"test": "https://gridproxy.test.grid.tf/",
		"qa":   "https://gridproxy.qa.grid.tf/",
		"main": "https://gridproxy.grid.tf/",
	}
	RelayURLs = map[string]string{
		"dev": "wss://relay.dev.grid.tf",
	}
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string, st state.StateI) (func() *schema.Provider, subi.SubstrateExt) {
	var substrateConnection subi.SubstrateExt
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"mnemonics": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("MNEMONICS", nil),
				},
				"key_type": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "key type registered on substrate (ed25519 or sr25519)",
					DefaultFunc: schema.EnvDefaultFunc("KEY_TYPE", "sr25519"),
				},
				"network": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "grid network, one of: dev test qa main",
					DefaultFunc: schema.EnvDefaultFunc("NETWORK", "dev"),
				},
				"substrate_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "substrate url, example: wss://tfchain.dev.grid.tf/ws",
					DefaultFunc: schema.EnvDefaultFunc("SUBSTRATE_URL", nil),
				},
				"rmb_redis_url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("RMB_URL", "tcp://127.0.0.1:6379"),
				},
				"rmb_proxy_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "rmb proxy url, example: https://gridproxy.dev.grid.tf/",
					DefaultFunc: schema.EnvDefaultFunc("RMB_PROXY_URL", nil),
				},
				"relay_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "rmb proxy url, example: wss://relay.dev.grid.tf",
					DefaultFunc: schema.EnvDefaultFunc("RELAY_URL", nil),
				},
				"use_rmb_proxy": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "whether to use the rmb proxy or not",
					DefaultFunc: schema.EnvDefaultFunc("USE_RMB_PROXY", true),
				},
				"verify_reply": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "whether to verify rmb replies (temporary for dev use only)",
					DefaultFunc: schema.EnvDefaultFunc("VERIFY_REPLY", false),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"grid_gateway_domain": dataSourceGatewayDomain(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"grid_scheduler":  ReourceScheduler(),
				"grid_deployment": resourceDeployment(),
				"grid_network":    resourceNetwork(),
				"grid_kubernetes": resourceKubernetes(),
				"grid_name_proxy": resourceGatewayNameProxy(),
				"grid_fqdn_proxy": resourceGatewayFQDNProxy(),
			},
		}
		configFunc, sub := providerConfigure(st)
		substrateConnection = sub
		p.ConfigureContextFunc = configFunc

		return p
	}, substrateConnection
}

type apiClient struct {
	twin_id       uint32
	mnemonics     string
	substrate_url string
	rmb_redis_url string
	use_rmb_proxy bool
	grid_client   proxy.Client
	rmb           rmb.Client
	substrateConn subi.SubstrateExt
	manager       subi.Manager
	identity      substrate.Identity
	state         state.StateI
}

func providerConfigure(st state.StateI) (func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics), subi.SubstrateExt) {
	var substrateConn subi.SubstrateExt
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		rand.Seed(time.Now().UnixNano())
		var err error
		apiClient := apiClient{}
		apiClient.mnemonics = d.Get("mnemonics").(string)
		key_type := d.Get("key_type").(string)
		var identity substrate.Identity
		if key_type == "ed25519" {
			identity, err = substrate.NewIdentityFromEd25519Phrase(string(apiClient.mnemonics))
		} else if key_type == "sr25519" {
			identity, err = substrate.NewIdentityFromSr25519Phrase(string(apiClient.mnemonics))
		} else {
			err = errors.New("key_type must be one of ed25519 and sr25519")
		}
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "error getting identity"))
		}
		sk, err := identity.KeyPair()
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "error getting user secret"))
		}
		apiClient.identity = identity
		network := d.Get("network").(string)
		if network != "dev" && network != "qa" && network != "test" && network != "main" {
			return nil, diag.Errorf("network must be one of dev, qa, test, and main")
		}
		apiClient.substrate_url = SUBSTRATE_URL[network]
		rmb_proxy_url := RMB_PROXY_URL[network]
		substrate_url := d.Get("substrate_url").(string)
		passed_rmb_proxy_url := d.Get("rmb_proxy_url").(string)
		if substrate_url != "" {
			log.Printf("substrate url is not null %s", substrate_url)
			apiClient.substrate_url = substrate_url
		}
		if passed_rmb_proxy_url != "" {
			rmb_proxy_url = passed_rmb_proxy_url
		}
		log.Printf("substrate url: %s %s\n", apiClient.substrate_url, substrate_url)
		apiClient.manager = subi.NewManager(apiClient.substrate_url)
		subx, err := apiClient.manager.SubstrateExt()
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
		}
		// substrate connection will be returned and closed in main.go
		substrateConn = subx
		apiClient.substrateConn = subx
		apiClient.use_rmb_proxy = d.Get("use_rmb_proxy").(bool)

		apiClient.rmb_redis_url = d.Get("rmb_redis_url").(string)

		if err := validateAccount(&apiClient, apiClient.substrateConn); err != nil {
			return nil, diag.FromErr(err)
		}
		pub := sk.Public()
		twin, err := subx.GetTwinByPubKey(pub)
		if err != nil && errors.Is(err, substrate.ErrNotFound) {
			return nil, diag.Errorf("no twin associated with the accound with the given mnemonics")
		}
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "failed to get twin for the given mnemonics"))
		}
		apiClient.twin_id = twin
		var cl rmb.Client

		sessionID := generateSessionID()

		sub, err := apiClient.manager.Substrate()
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "failed to get substrate client"))
		}
		relayURL := d.Get("relay_url").(string)
		if relayURL != "" {
			cl, err = direct.NewClient(context.Background(), key_type, apiClient.mnemonics, relayURL, sessionID, sub, false)
		} else {
			relayURL, ok := RelayURLs[network]
			if !ok {
				return nil, diag.Errorf("error getting relay url for network %s", network)
			}

			cl, err = direct.NewClient(context.Background(), key_type, apiClient.mnemonics, relayURL, sessionID, sub, false)
		}

		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "couldn't create rmb client"))
		}
		apiClient.rmb = cl

		grid_client := proxy.NewClient(rmb_proxy_url)
		apiClient.grid_client = proxy.NewRetryingClient(grid_client)
		if err := preValidate(&apiClient, subx); err != nil {
			return nil, diag.FromErr(err)
		}
		apiClient.state = st
		return &apiClient, nil
	}, substrateConn
}

func generateSessionID() string {
	return fmt.Sprintf("tf-%d", os.Getpid())
}
