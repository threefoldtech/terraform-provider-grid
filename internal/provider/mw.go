package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

type Marshalable interface {
	Marshal(d *schema.ResourceData)
	sync(ctx context.Context, sub subi.SubstrateExt) (err error)
}

type Action func(context.Context, subi.SubstrateExt, *schema.ResourceData, *apiClient) (Marshalable, error)

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
					Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
					Detail:   diags[idx].Summary,
				}
			}
		}
		return diags
	}
}

func resourceFunc(a Action, reportSync bool) func(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, i interface{}) (diags diag.Diagnostics) {
		cl := i.(*apiClient)
		rmbctx, cancel := context.WithCancel(ctx)
		defer cancel()
		go startRmbIfNeeded(rmbctx, cl)
		sub, err := cl.manager.SubstrateExt()
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
		}
		defer sub.Close()
		obj, err := a(ctx, sub, d, cl)
		if err != nil {
			diags = diag.FromErr(err)
		}
		if obj != nil {
			if err := obj.sync(ctx, sub); err != nil && reportSync {
				diags = append(diags, diag.FromErr(err)...)
			}
			obj.Marshal(d)
		}
		return diags
	}
}
