package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDisk() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample data source in the Terraform provider scaffolding.",

		ReadContext: dataSourceDiskRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "Disk ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "Disk Name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"version": {
				Description: "Version",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"description": {
				Description: "Description field",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"size": {
				Description: "Disk size in Gigabytes",
				Type:        schema.TypeInt,
				Computed:    true,
			},
		},
	}
}

func dataSourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	idFromAPI := "my-id"
	d.SetId(idFromAPI)

	return diag.Errorf("not implemented")
}
