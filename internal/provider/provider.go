package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/vedhavyas/go-subkey"
)

var (
	SUBSTRATE_URL = map[string]string{
		"dev":  "wss://tfchain.dev.threefold.io/ws",
		"test": "wss://tfchain.test.threefold.io/ws",
	}
	GRAPHQL_URL = map[string]string{
		"dev":  "https://tfchain.dev.threefold.io/graphql/graphql/",
		"test": "https://tfchain.test.threefold.io/graphql/graphql/",
	}
	RMB_PROXY_URL = map[string]string{
		"dev":  "https://rmbproxy1.devnet.grid.tf/",
		"test": "https://rmbproxy1.testnet.grid.tf/",
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

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"mnemonics": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("MNEMONICS", nil),
				},
				"network": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "grid network, one of: dev test",
					DefaultFunc: schema.EnvDefaultFunc("NETWORK", "dev"),
				},
				"substrate_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "substrate url, example: wss://tfchain.dev.threefold.io/ws",
					DefaultFunc: schema.EnvDefaultFunc("SUBSTRATE_URL", nil),
				},
				"graphql_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "graphql url, example: https://tfchain.dev.threefold.io/graphql/graphql/",
					DefaultFunc: schema.EnvDefaultFunc("GRAPHQL_URL", nil),
				},
				"rmb_redis_url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("RMB_URL", "tcp://127.0.0.1:6379"),
				},
				"rmb_proxy_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "rmb proxy url, example: https://rmbproxy1.devnet.grid.tf/",
					DefaultFunc: schema.EnvDefaultFunc("RMB_PROXY_URL", nil),
				},
				"use_rmb_proxy": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "whether to use the rmb proxy or not",
					DefaultFunc: schema.EnvDefaultFunc("USE_RMB_PROXY", true),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"grid_gateway_domain": dataSourceGatewayDomain(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"grid_deployment": resourceDeployment(),
				"grid_network":    resourceNetwork(),
				"grid_kubernetes": resourceKubernetes(),
				"grid_name_proxy": resourceGatewayNameProxy(),
				"grid_fqdn_proxy": resourceGatewayFQDNProxy(),
			},
		}

		p.ConfigureContextFunc = providerConfigure

		return p
	}
}

type apiClient struct {
	twin_id       uint32
	mnemonics     string
	graphql_url   string
	substrate_url string
	rmb_redis_url string
	use_rmb_proxy bool
	rmb_proxy_url string
	userSK        subkey.KeyPair
	rmb           rmb.Client
	sub           *substrate.Substrate
	identity      *substrate.Identity
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var err error

	apiClient := apiClient{}
	apiClient.mnemonics = d.Get("mnemonics").(string)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "error getting identity"))
	}
	sk, err := identity.SecureKey()
	apiClient.userSK = sk
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "error getting user secret"))
	}
	apiClient.identity = &identity
	network := d.Get("network").(string)
	if network != "dev" && network != "test" {
		return nil, diag.Errorf("network must be one of dev and test")
	}
	apiClient.substrate_url = SUBSTRATE_URL[network]
	apiClient.graphql_url = GRAPHQL_URL[network]
	apiClient.rmb_proxy_url = RMB_PROXY_URL[network]
	substrate_url := d.Get("substrate_url").(string)
	graphql_url := d.Get("graphql_url").(string)
	rmb_proxy_url := d.Get("rmb_proxy_url").(string)
	if substrate_url != "" {
		log.Printf("substrate url is not null %s", substrate_url)
		apiClient.substrate_url = substrate_url
	}
	if graphql_url != "" {
		apiClient.graphql_url = graphql_url
	}
	if rmb_proxy_url != "" {
		apiClient.rmb_proxy_url = rmb_proxy_url
	}
	log.Printf("substrate url: %s %s\n", apiClient.substrate_url, substrate_url)
	apiClient.sub, err = substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "couldn't create substrate client"))
	}
	apiClient.use_rmb_proxy = d.Get("use_rmb_proxy").(bool)

	apiClient.rmb_redis_url = d.Get("rmb_redis_url").(string)

	if err := validateAccount(&apiClient); err != nil {
		return nil, diag.FromErr(err)
	}
	pub := sk.Public()
	twin, err := apiClient.sub.GetTwinByPubKey(pub)
	if err != nil && errors.Is(err, substrate.ErrNotFound) {
		return nil, diag.Errorf("no twin associated with the accound with the given mnemonics")
	}
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "failed to get twin for the given mnemonics"))
	}
	apiClient.twin_id = twin
	var cl rmb.Client
	if apiClient.use_rmb_proxy {
		cl = client.NewProxyBus(apiClient.rmb_proxy_url, apiClient.twin_id)
	} else {
		cl, err = rmb.NewClient(apiClient.rmb_redis_url)
	}
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "couldn't create rmb client"))
	}
	apiClient.rmb = cl
	if err := preValidate(&apiClient); err != nil {
		return nil, diag.FromErr(err)
	}
	return &apiClient, nil
}
