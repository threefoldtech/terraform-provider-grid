package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "group resource",
		ReadContext: resourceGroupRead,

		Schema: map[string]*schema.Schema{
			"group_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "group id",
			},
		},
	}
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cl := meta.(apiClient)
	var diags diag.Diagnostics
	group_id, err := strconv.ParseUint(d.Id(), 10, 32)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	_, err = cl.substrateConn.GetGroup(group_id)
	if err != nil && errors.Is(err, substrate.ErrNotFound) {
		d.SetId("")
	} else if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	return nil
}

