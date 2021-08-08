package provider

import (
	"context"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/zos/pkg/rmb"
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
				"twin_id": &schema.Schema{
					Type:        schema.TypeInt,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("TWIN_ID", 0),
				},
				"seed": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("SEED", nil),
				},
				"substrate_url": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("SUBSTRATE_URL", "wss://explorer.devnet.grid.tf/ws"),
				},
				"rmb_url": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("RMB_URL", "tcp://127.0.0.1:6379"),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"scaffolding_data_source": dataSourceDisk(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"grid_disk": resourceDisk(),
			},
		}

		p.ConfigureContextFunc = providerConfigure

		return p
	}
}

type apiClient struct {
	twin_id       uint32
	seed          []byte
	substrate_url string
	rmb_url       string
	client        rmb.Client
}

// func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
// 	return func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
// 		// Setup a User-Agent for your API client (replace the provider name for yours):
// 		// userAgent := p.UserAgent("terraform-provider-scaffolding", version)
// 		myClient = apiClient()
// 		p.user

// 		return &apiClient{}, nil
// 	}
// }

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	seed, err := hex.DecodeString(d.Get("seed").(string))
	if err != nil {
		panic(err)
	}
	apiClient := apiClient{}
	apiClient.twin_id = uint32(d.Get("twin_id").(int))
	apiClient.substrate_url = d.Get("substrate_url").(string)
	apiClient.rmb_url = d.Get("rmb_url").(string)
	apiClient.seed = seed
	cl, err := rmb.NewClient(apiClient.rmb_url)

	if err != nil {
		panic(err)
	}
	apiClient.client = cl

	return &apiClient, nil
}
