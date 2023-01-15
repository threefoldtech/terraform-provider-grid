// Package provider is the terraform provider
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

const errTerraformOutSync = "Error reading data from remote, terraform state might be out of sync with the remote state"

// Syncer struct
type Syncer interface {
	SyncContractsDeployments(d *schema.ResourceData) (err error)
	Sync(ctx context.Context, sub subi.SubstrateExt, cl *threefoldPluginClient) (err error)
}

// Action is a type for a function
type Action func(context.Context, subi.SubstrateExt, *schema.ResourceData, *threefoldPluginClient) (Syncer, error)

func ResourceFunc(a Action) func(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, i interface{}) (diags diag.Diagnostics) {
		return resourceFunc(a, false)(ctx, d, i)
	}
}
func ResourceReadFunc(a Action) func(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, i interface{}) (diags diag.Diagnostics) {
		diags = resourceFunc(a, true)(ctx, d, i)
		if diags.HasError() {
			for idx := range diags {
				diags[idx] = diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  errTerraformOutSync,
					Detail:   diags[idx].Summary,
				}
			}
		}
		return diags
	}
}

func resourceFunc(a Action, reportSync bool) func(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, i interface{}) (diags diag.Diagnostics) {
		cl := i.(*threefoldPluginClient)
		if err := validateAccountBalanceForExtrinsics(cl.substrateConn, cl.identity); err != nil {
			return diag.FromErr(err)
		}

		obj, err := a(ctx, cl.substrateConn, d, cl)
		if err != nil {
			diags = diag.FromErr(err)
		}
		if obj != nil {
			if err := obj.Sync(ctx, cl.substrateConn, cl); err != nil {
				if reportSync {
					diags = append(diags, diag.FromErr(err)...)
				} else {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "failed to read deployment data (terraform refresh might help)",
						Detail:   err.Error(),
					})
				}
			}
			err = obj.SyncContractsDeployments(d)
			if err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}
		return diags
	}
}
