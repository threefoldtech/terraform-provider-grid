// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/grid3-go/deployer"
)

func resourceGatewayNameProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description:   "Resource for deploying a gateway name workload. A user should specify some unique name, for example hamada, and a node working as a gateway that has the domain gent01.dev.grid.tf, and the grid generates a fully qualified domain name (fqdn) `hamada.getn01.dev.grid.tf`. Then, the user could connect this gateway workload to whichever backend services the user desires, making these backend services accessible through the computed fqdn.",
		CreateContext: resourceGatewayNameCreate,
		ReadContext:   resourceGatewayNameRead,
		UpdateContext: resourceGatewayNameUpdate,
		DeleteContext: resourceGatewayNameDelete,

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
			"network": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Network name to join, if backend IP is private.",
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

func resourceGatewayNameCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newNameGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load name gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Deploy(ctx, gw); err != nil {
		return diag.Errorf("couldn't deploy name gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync name gateway with error: %v", err)
	}

	if err := syncContractsNameGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set name gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayNameUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newNameGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load name gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Deploy(ctx, gw); err != nil {
		return diag.Errorf("couldn't update name gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync name gateway with error: %v", err)
	}

	if err := syncContractsNameGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set name gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayNameRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newNameGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load name gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Sync(ctx, gw); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "failed to read deployment data (terraform refresh might help)",
			Detail:   err.Error(),
		})
		return diags
	}

	if err := syncContractsNameGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set name gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayNameDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newNameGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load name gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Cancel(ctx, gw); err != nil {
		return diag.Errorf("couldn't cancel name gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayNameDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync name gateway with error: %v", err)
	}

	if err := syncContractsNameGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set name gateway data to the resource with error: %v", err)
	}

	return diags
}
