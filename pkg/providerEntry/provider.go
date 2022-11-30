package providerentry

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	provider "github.com/threefoldtech/terraform-provider-grid/internal/provider"
)

func New(version string, st state.StateI) (func() *schema.Provider, *substrate.Substrate) {
	var substrateConnection *substrate.Substrate
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
				"grid_gateway_domain": provider.DataSourceGatewayDomain(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"grid_scheduler":         provider.ReourceScheduler(),
				"grid_deployment":        provider.ResourceDeployment(),
				"grid_network":           provider.ResourceNetwork(),
				"grid_kubernetes":        provider.ResourceKubernetes(),
				"grid_name_proxy":        provider.ResourceGatewayNameProxy(),
				"grid_fqdn_proxy":        provider.ResourceGatewayFQDNProxy(),
				"grid_capacity_reserver": provider.ResourceCapacityReserver(),
			},
		}
		configFunc, sub := provider.ProviderConfigure(st)
		substrateConnection = sub
		p.ConfigureContextFunc = configFunc

		return p
	}, substrateConnection
}
