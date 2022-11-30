package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceCapacityReserver() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for computing deployments requirements.",

		ReadContext:   resourceDeploymentsRead,
		CreateContext: resourceDeploymentsCreate,
		UpdateContext: resourceDeploymentsUpdate,
		DeleteContext: resourceDeploymentsDelete,

		Schema: map[string]*schema.Schema{
			"nodes": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of nodes to add to the network",
			},
			"deployments": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"farm": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Farm id of deployment",
						},
						"node": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Node id to place the deployment on",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Number of VCPUs",
						},
						"memory": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory size",
						},
					},
				},
			},
		},
	}
}

func resourceDeploymentsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
func resourceDeploymentsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	// this will call reserve capacity and set node ids in deployments

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}
func resourceDeploymentsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	if d.HasChange("deployments") {
		tflog.Info(ctx, "deployment has changed")
		oldValues, newValues := d.GetChange("deployments")
		tflog.Info(ctx, "old deployments")
		oldValuesList := oldValues.([]interface{})
		newValuesList := newValues.([]interface{})
		// the user may add new deployments, so we need to reserve capacity for it
		var newCapacityRequests []interface{}
		// update capacity for old deployments if changed
		var updateCapacityRequests []interface{}

		tflog.Info(ctx, "new deployments")
		for idx, deployment := range newValuesList {
			// check if any of old deployments got updated
			if len(oldValuesList) >= idx+1 {
				if deployment.(map[string]interface{})["cpu"].(int) != oldValuesList[idx].(map[string]interface{})["cpu"].(int) ||
					deployment.(map[string]interface{})["memory"].(int) != oldValuesList[idx].(map[string]interface{})["memory"].(int) {
					updateCapacityRequests = append(updateCapacityRequests, deployment)
				}
				continue
			}
			// if new deployments append to newCapacityRequests
			newCapacityRequests = append(newCapacityRequests, deployment)
		}
		tflog.Info(ctx, fmt.Sprintf("newCapacityRequests: %d ------------------ oldCapacityRequests: %d", len(newCapacityRequests), len(updateCapacityRequests)))
	}
	// this will call update capacity and set node ids
	return nil
}
func resourceDeploymentsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
