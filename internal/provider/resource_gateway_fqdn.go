// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/grid3-go/deployer"
)

func resourceGatewayFQDNProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying a gateway with a fully qualified domain name.\nA user should have some fully qualified domain name (fqdn) (e.g. example.com), pointing to the specified node working as a gateway, then connect this gateway to whichever backend services they desire, making these backend services accessible through the computed fqdn.",

		CreateContext: resourceGatewayFQDNCreate,
		ReadContext:   resourceGatewayFQDNRead,
		UpdateContext: resourceGatewayFQDNUpdate,
		DeleteContext: resourceGatewayFQDNDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "name",
				Description: "Gateway workload name.  This has to be unique within the deployment.",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Solution type for created contract to be consistent across threefold tooling.",
				Default:     "Gateway",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the gateway fqdn workload.",
			},
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The gateway's node id.",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The fully qualified domain name of the deployed workload.",
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
		},
	}
}

func resourceGatewayFQDNCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newFQDNGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load fqdn gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Deploy(ctx, gw); err != nil {
		return diag.Errorf("couldn't deploy fqdn gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync fqdn gateway with error: %v", err)
	}

	if err := syncContractsFQDNGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set fqdn gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayFQDNUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newFQDNGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load fqdn gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Deploy(ctx, gw); err != nil {
		return diag.Errorf("couldn't update fqdn gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync fqdn gateway with error: %v", err)
	}

	if err := syncContractsFQDNGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set fqdn gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayFQDNRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newFQDNGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load fqdn gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Sync(ctx, gw); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "failed to read deployment data (terraform refresh might help)",
			Detail:   err.Error(),
		})
		return diags
	}

	if err := syncContractsFQDNGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set fqdn gateway data to the resource with error: %v", err)
	}

	return diags
}

func resourceGatewayFQDNDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	gw, err := newFQDNGatewayFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load fqdn gateway data with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Cancel(ctx, gw); err != nil {
		return diag.Errorf("couldn't update fqdn gateway with error: %v", err)
	}

	if err := tfPluginClient.GatewayFQDNDeployer.Sync(ctx, gw); err != nil {
		return diag.Errorf("couldn't sync fqdn gateway with error: %v", err)
	}

	if err := syncContractsFQDNGateways(d, gw); err != nil {
		return diag.Errorf("couldn't set fqdn gateway data to the resource with error: %v", err)
	}

	return diags
}
