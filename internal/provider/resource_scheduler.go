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
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func ReourceScheduler() *schema.Resource {
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
						"farm": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Farm name to search for eligible nodes.",
						},
						"ipv4": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick only nodes with public ipv4 configuration.",
						},
						"domain": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick only nodes with public config containing domain.",
						},
						"certified": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to pick only certified nodes (Not implemented).",
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
		assignment[k] = v.(uint32)
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
		reqs = append(reqs, scheduler.Request{
			Name:      mp["name"].(string),
			Farm:      mp["farm"].(string),
			HasIPv4:   mp["ipv4"].(bool),
			HasDomain: mp["domain"].(bool),
			Certified: mp["certified"].(bool),
			Capacity: scheduler.Capacity{
				MRU: uint64(mp["mru"].(int)) * uint64(gridtypes.Megabyte),
				HRU: uint64(mp["hru"].(int)) * uint64(gridtypes.Megabyte),
				SRU: uint64(mp["sru"].(int)) * uint64(gridtypes.Megabyte),
			},
		})
	}
	return reqs
}

func schedule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	threefoldPluginClient, ok := meta.(*threefoldPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	assignment := parseAssignment(d)
	reqs := parseRequests(d, assignment)
	scheduler := scheduler.NewScheduler(threefoldPluginClient.gridProxyClient, uint64(threefoldPluginClient.twinID))
	for _, r := range reqs {
		node, err := scheduler.Schedule(&r)
		if err != nil {
			return diag.FromErr(errors.Wrapf(err, "couldn't schedule request %s", r.Name))
		}
		assignment[r.Name] = node
	}
	err := d.Set("nodes", assignment)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "couldn't set nodes with %v", assignment))
	}
	return nil

}

func ResourceSchedRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

func ResourceSchedCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	diags := schedule(ctx, d, meta)
	if diags.HasError() {
		return diags
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}

func ResourceSchedUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return schedule(ctx, d, meta)
}

func ResourceSchedDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{}
}
