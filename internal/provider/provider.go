package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/substrate"
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
				"twin_id": {
					Type:        schema.TypeInt,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("TWIN_ID", 0),
				},
				"mnemonics": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("MNEMONICS", nil),
				},
				"substrate_url": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("SUBSTRATE_URL", "wss://explorer.devnet.grid.tf/ws"),
				},
				"rmb_url": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("RMB_URL", "tcp://127.0.0.1:6379"),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"scaffolding_data_source": dataSourceDisk(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"grid_deployment": resourceDeployment(),
				"grid_network":    resourceNetwork(),
				"grid_kubernetes": resourceKubernetes(),
			},
		}

		p.ConfigureContextFunc = providerConfigure

		return p
	}
}

type apiClient struct {
	twin_id       uint32
	mnemonics     string
	substrate_url string
	rmb_url       string
	rmb           rmb.Client
	sub           *substrate.Substrate
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

	apiClient := apiClient{}
	apiClient.mnemonics = d.Get("mnemonics").(string)
	apiClient.twin_id = uint32(d.Get("twin_id").(int))
	apiClient.substrate_url = d.Get("substrate_url").(string)
	apiClient.rmb_url = d.Get("rmb_url").(string)
	cl, err := rmb.NewClient(apiClient.rmb_url)

	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "couldn't create rmb client"))
	}
	apiClient.rmb = cl
	apiClient.sub, err = substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "couldn't create substrate client"))
	}

	return &apiClient, nil
}
