// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/grid3-go/deployer"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func ResourceScheduler() *schema.Resource {
	return &schema.Resource{
		Description:   "Resource to dynamically assign resource requests to nodes. A user could specify their desired node configurations, and the scheduler searches the grid for eligible nodes.",
		CreateContext: ResourceSchedCreate,
		UpdateContext: ResourceSchedUpdate,
		ReadContext:   ResourceSchedRead,
		DeleteContext: ResourceSchedDelete,
		Schema: map[string]*schema.Schema{
			"requests": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of requests. Here a user defines their required nodes configurations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Request name. Used as a reference in the `nodes` dict.",
						},
						"cru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of required virtual CPUs.",
						},
						"mru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory size in MBs.",
						},
						"sru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Disk SSD size in MBs.",
						},
						"hru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Disk HDD size in MBs.",
						},
						"farm_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Farm id to search for eligible nodes.",
						},
						"public_config": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick only nodes with public config containing domain.",
						},
						"public_ips_count": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Required count of public ips.",
						},
						"certified": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick only certified nodes (Not implemented).",
						},
						"dedicated": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick a rentable node",
						},
						"node_exclude": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeInt,
							},
							Description: "List of node ids you want to exclude from the search.",
						},
						"distinct": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "True to ensure this request returns a distinct node relative to this scheduler resource.",
						},
					},
				},
			},
			"nodes": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from the request name to the node id.",
			},
		},
	}
}

func parseAssignment(d *schema.ResourceData) map[string]uint32 {
	assignmentIfs := d.Get("nodes").(map[string]interface{})
	assignment := make(map[string]uint32)
	for k, v := range assignmentIfs {
		assignment[k] = uint32(v.(int))
	}
	return assignment
}

func parseRequests(d *schema.ResourceData, assignment map[string]uint32) []scheduler.Request {
	reqsIfs := d.Get("requests").([]interface{})
	reqs := make([]scheduler.Request, 0)
	for _, r := range reqsIfs {
		mp := r.(map[string]interface{})
		if _, ok := assignment[mp["name"].(string)]; ok {
			// skip already assigned ones
			continue
		}
		nodesToExcludeIF := mp["node_exclude"].([]interface{})
		nodesToExclude := make([]uint32, len(nodesToExcludeIF))
		for idx, n := range nodesToExcludeIF {
			nodesToExclude[idx] = uint32(n.(int))
		}

		reqs = append(reqs, scheduler.Request{
			Name:           mp["name"].(string),
			FarmId:         uint32(mp["farm_id"].(int)),
			PublicConfig:   mp["public_config"].(bool),
			PublicIpsCount: uint32(mp["public_ips_count"].(int)),
			Certified:      mp["certified"].(bool),
			Dedicated:      mp["dedicated"].(bool),
			NodeExclude:    nodesToExclude,
			Capacity: scheduler.Capacity{
				MRU: uint64(mp["mru"].(int)) * uint64(gridtypes.Megabyte),
				HRU: uint64(mp["hru"].(int)) * uint64(gridtypes.Megabyte),
				SRU: uint64(mp["sru"].(int)) * uint64(gridtypes.Megabyte),
			},
			Distinct: mp["distinct"].(bool),
		})
	}
	return reqs
}

func schedule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}
	// read previously assigned nodes
	assignment := parseAssignment(d)
	reqs := parseRequests(d, assignment)

	scheduler := scheduler.NewScheduler(tfPluginClient.GridProxyClient, uint64(tfPluginClient.TwinID), tfPluginClient.RMB)
	if err := scheduler.ProcessRequests(ctx, reqs, assignment); err != nil {
		return diag.FromErr(err)
	}

	err := d.Set("nodes", assignment)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "couldn't set nodes with %v", assignment))
	}
	return nil

}

// ResourceSchedRead reads for schedule resource
func ResourceSchedRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

// ResourceSchedCreate creates for schedule resource
func ResourceSchedCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	diags := schedule(ctx, d, meta)
	if diags.HasError() {
		return diags
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}

// ResourceSchedUpdate updates for schedule resource
func ResourceSchedUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return schedule(ctx, d, meta)
}

// ResourceSchedDelete deletes for schedule resource
func ResourceSchedDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{}
}
