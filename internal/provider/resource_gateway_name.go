// Package provider is the terraform provider
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func resourceGatewayNameProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying a gateway name workload. A user should specify some unique name, for example hamada, and a node working as a gateway that has the domain gent01.dev.grid.tf, and the grid generates a fully qualified domain name (fqdn) `hamada.getn01.dev.grid.tf`. Then, the user could connect this gateway workload to whichever backend services the user desires, making these backend services accessible through the computed fqdn.",

		CreateContext: ResourceFunc(resourceGatewayNameCreate),
		ReadContext:   ResourceReadFunc(resourceGatewayNameRead),
		UpdateContext: ResourceFunc(resourceGatewayNameUpdate),
		DeleteContext: ResourceFunc(resourceGatewayNameDelete),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Domain prefix. The fqdn will be <name>.<gateway-domain>.  This has to be unique within the deployment.",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Solution type for created contract to be consistent across threefold tooling.",
				Default:     "Gateway",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Description of the gateway name workload.",
			},
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The gateway's node id.",
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
				Description: "TLS passthrough controls the TLS termination, if false, the gateway will terminate the TLS, if True, it will only be terminated by the backend service.",
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
			"name_contract_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The id of the created name contract.",
			},
		},
	}
}

func resourceGatewayNameCreate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, threefoldPluginClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayNameUpdate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, threefoldPluginClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceGatewayNameRead(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, threefoldPluginClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, nil
}

func resourceGatewayNameDelete(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (Marshalable, error) {
	deployer, err := NewGatewayNameDeployer(d, threefoldPluginClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}
	return &deployer, deployer.Cancel(ctx, sub)
}
