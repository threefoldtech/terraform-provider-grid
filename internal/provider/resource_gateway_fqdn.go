// Package provider is the terraform provider
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func resourceGatewayFQDNProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying a gateway with a fully qualified domain name.\nA user should have some fully qualified domain name (fqdn) (e.g. example.com), pointing to the specified node working as a gateway, then connect this gateway to whichever backend services they desire, making these backend services accessible through the computed fqdn.",

		CreateContext: ResourceFunc(resourceGatewayFQDNCreate),
		ReadContext:   ResourceReadFunc(resourceGatewayFQDNRead),
		UpdateContext: ResourceFunc(resourceGatewayFQDNUpdate),
		DeleteContext: ResourceFunc(resourceGatewayFQDNDelete),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "name",
				Description: "Gateway workload name.",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Solution type for created contract, displayed [here](https://play.dev.grid.tf/#/contractslist).",
				Default:     "Gateway",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description for the gateway fqdn workload.",
			},
			"node_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The gateway's node id.",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The fully quallified domain name of the deployed workload.",
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
				Description: "The backends of the gateway proxy (in the format (http|https)://ip:port), with tls_passthrough the scheme must be https.",
			},
			"node_deployment_id": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each node to its deployment id.",
			},
		},
	}
}

func resourceGatewayFQDNCreate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayFQDNUpdate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayFQDNRead(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, nil
}

func resourceGatewayFQDNDelete(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Cancel(ctx, sub)
}
