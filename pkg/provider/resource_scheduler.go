package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/provider/scheduler"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func ReourceScheduler() *schema.Resource {
	return &schema.Resource{
		// TODO: update descriptions
		Description:   "Resource to dynamically assign resource requests to nodes.",
		CreateContext: ReourceSchedCreate,
		UpdateContext: ReourceSchedUpdate,
		ReadContext:   ReourceSchedRead,
		DeleteContext: ReourceSchedDelete,
		Schema: map[string]*schema.Schema{
			"requests": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of node assignment requests",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "used as a key in the `nodes` dict to be used as a reference",
						},
						"cru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Number of VCPUs",
						},
						"mru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory size in MBs",
						},
						"sru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Disk SSD size in MBs",
						},
						"hru": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Disk HDD size in MBs",
						},
						"farm": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Farm name",
						},
						"ipv4": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Pick only nodes with public config containing ipv4",
						},
						"domain": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Pick only nodes with public config containing domain",
						},
						"certified": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Pick only certified nodes (Not implemented)",
						},
					},
				},
			},
			"nodes": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from the request name to the node id",
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
			Cap: scheduler.Capacity{
				Memory: uint64(mp["mru"].(int)) * uint64(gridtypes.Megabyte),
				Hru:    uint64(mp["hru"].(int)) * uint64(gridtypes.Megabyte),
				Sru:    uint64(mp["sru"].(int)) * uint64(gridtypes.Megabyte),
			},
		})
	}
	return reqs
}

func schedule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	assignment := parseAssignment(d)
	reqs := parseRequests(d, assignment)
	scheduler := scheduler.NewScheduler(apiClient.grid_client, uint64(apiClient.twin_id))
	for _, r := range reqs {
		node, err := scheduler.Schedule(&r)
		if err != nil {
			return diag.FromErr(errors.Wrapf(err, "couldn't schedule request %s", r.Name))
		}
		assignment[r.Name] = node
	}
	d.Set("nodes", assignment)
	return nil

}

func ReourceSchedRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

func ReourceSchedCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	diags := schedule(ctx, d, meta)
	if diags.HasError() {
		return diags
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}

func ReourceSchedUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return schedule(ctx, d, meta)
}

func ReourceSchedDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{}
}
