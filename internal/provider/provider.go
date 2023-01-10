// Package provider is the terraform provider
package provider

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/rmb"
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
	SubstrateVersion = map[string]func(url ...string) subi.Manager{
		"dev":  subi.NewDevManager,
		"qa":   subi.NewQAManager,
		"test": subi.NewTestManager,
		"main": subi.NewMainManager,
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

type threefoldPluginClient struct {
	twinID          uint32
	mnemonics       string
	substrateURL    string
	rmbRedisURL     string
	useRmbProxy     bool
	gridProxyClient proxy.Client
	rmb             rmb.Client
	substrateConn   subi.SubstrateExt
	manager         subi.Manager
	identity        subi.Identity
	state           state.StateI
}

func providerConfigure(st state.StateI) (func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics), subi.SubstrateExt) {
	var substrateConn subi.SubstrateExt
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var err error
		threefoldPluginClient := threefoldPluginClient{}
		threefoldPluginClient.mnemonics = d.Get("mnemonics").(string)
		if err := validateMnemonics(threefoldPluginClient.mnemonics); err != nil {
			return nil, diag.FromErr(err)
		}

		keyType := d.Get("key_type").(string)
		var identity subi.Identity

		switch keyType {
		case "ed25519":
			identity, err = subi.NewIdentityFromEd25519Phrase(string(threefoldPluginClient.mnemonics))
		case "sr25519":
			identity, err = subi.NewIdentityFromSr25519Phrase(string(threefoldPluginClient.mnemonics))
		default:
			err = errors.New("key_type must be one of ed25519 and sr25519")
		}

		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "error getting identity"))
		}

		keyPair, err := identity.KeyPair()
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "error getting user secret"))
		}

		threefoldPluginClient.identity = identity
		network := d.Get("network").(string)
		if network != "dev" && network != "qa" && network != "test" && network != "main" {
			return nil, diag.Errorf("network must be one of dev, qa, test, and main")
		}
		threefoldPluginClient.substrateURL = SUBSTRATE_URL[network]
		rmbProxyURL := RMB_PROXY_URL[network]
		substrateURL := d.Get("substrate_url").(string)
		passedRmbProxyURL := d.Get("rmb_proxy_url").(string)
		if len(strings.TrimSpace(substrateURL)) != 0 {
			log.Printf("substrate url is not null %s", substrateURL)
			threefoldPluginClient.substrateURL = substrateURL
		}
		if len(strings.TrimSpace(passedRmbProxyURL)) != 0 {
			rmbProxyURL = passedRmbProxyURL
		}

		log.Printf("substrate url: %s %s\n", threefoldPluginClient.substrateURL, substrateURL)
		threefoldPluginClient.manager = SubstrateVersion[network](threefoldPluginClient.substrateURL)
		sub, err := threefoldPluginClient.manager.SubstrateExt()
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
		}
		// substrate connection will be returned and closed in main.go
		substrateConn = sub
		threefoldPluginClient.substrateConn = sub
		threefoldPluginClient.useRmbProxy = d.Get("use_rmb_proxy").(bool)

		threefoldPluginClient.rmbRedisURL = d.Get("rmb_redis_url").(string)

		if err := validateAccount(&threefoldPluginClient, threefoldPluginClient.substrateConn); err != nil {
			return nil, diag.FromErr(err)
		}
		pub := keyPair.Public()
		twin, err := sub.GetTwinByPubKey(pub)
		if err != nil && errors.Is(err, subi.ErrNotFound) {
			return nil, diag.Errorf("no twin associated with the account with the given mnemonics")
		}
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "failed to get twin for the given mnemonics"))
		}

		threefoldPluginClient.twinID = twin
		var cl rmb.Client
		if threefoldPluginClient.useRmbProxy {
			verifyReply := d.Get("verify_reply").(bool)
			cl, err = client.NewProxyBus(rmbProxyURL, threefoldPluginClient.twinID, threefoldPluginClient.substrateConn, identity, verifyReply)
		} else {
			cl, err = rmb.NewRMBClient(threefoldPluginClient.rmbRedisURL)
		}
		if err != nil {
			return nil, diag.FromErr(errors.Wrap(err, "couldn't create rmb client"))
		}
		threefoldPluginClient.rmb = cl

		gridProxyClient := proxy.NewClient(rmbProxyURL)
		threefoldPluginClient.gridProxyClient = proxy.NewRetryingClient(gridProxyClient)
		if err := validateClientRMB(&threefoldPluginClient, sub); err != nil {
			return nil, diag.FromErr(err)
		}
		threefoldPluginClient.state = st
		return &threefoldPluginClient, nil
	}, substrateConn
}
