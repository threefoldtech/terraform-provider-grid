package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

func resourceGatewayNameProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying gateway domains.",

		CreateContext: ResourceFunc(resourceGatewayNameCreate),
		ReadContext:   ResourceReadFunc(resourceGatewayNameRead),
		UpdateContext: ResourceFunc(resourceGatewayNameUpdate),
		DeleteContext: ResourceFunc(resourceGatewayNameDelete),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Gateway name (the fqdn will be <name>.<gateway-domain>)",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Gateway name (the fqdn will be <name>.<gateway-domain>)",
				Default:     "Gateway",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"capacity_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Capacity reservation contract id from capacity reserver",
			},
			"node_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The gateway's node id",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The computed fully quallified domain name of the deployed workload.",
			},
			"tls_passthrough": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "True to pass the tls as is to the backends.",
			},
			"backends": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The backends of the gateway proxy (in the format (http|https)://ip:port), with tls_passthrough the scheme must be https",
			},
			"capacity_deployment_map": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each contract to its deployment id",
			},
			"name_contract_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The id of the name contract",
			},
		},
	}
}

func resourceGatewayNameCreate(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayNameUpdate(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayNameRead(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, nil
}

func resourceGatewayNameDelete(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Cancel(ctx, sub)
}
