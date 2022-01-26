package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	gridproxy "github.com/threefoldtech/terraform-provider-grid/internal/gridproxy"
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

type MachineCapacity struct {
	CPUs   uint64
	Memory uint64
	Sru    uint64
	Hru    uint64
}

type NodeData struct {
	ID        uint32
	Farm      string
	HasIPv4   bool
	HasDomain bool
	Certified bool
	Cap       MachineCapacity
}

type Request struct {
	Cap       MachineCapacity
	Name      string
	Farm      string
	HasIPv4   bool
	HasDomain bool
	Certified bool
}

func getFarms(client *gridproxy.GridProxyClient) (map[int]string, error) {
	farms, err := client.Farms()
	if err != nil {
		return nil, err
	}
	farmMap := make(map[int]string)
	for _, f := range farms {
		farmMap[f.FarmID] = f.Name
	}
	return farmMap, nil
}

func freeCapacity(client *gridproxy.GridProxyClient, nodeID uint32) (MachineCapacity, error) {
	var res MachineCapacity
	node, err := client.Node(nodeID)
	if err != nil {
		return res, errors.Wrapf(err, "couldn't fetch node %d", nodeID)
	}

	res.CPUs = node.Capacity.Total.CRU - node.Capacity.Used.CRU
	res.Memory = uint64(node.Capacity.Total.MRU) - uint64(node.Capacity.Used.MRU)
	res.Hru = uint64(node.Capacity.Total.HRU) - uint64(node.Capacity.Used.HRU)
	res.Sru = uint64(node.Capacity.Total.SRU) - uint64(node.Capacity.Used.SRU)

	return res, nil
}

func getNodes(client *gridproxy.GridProxyClient) ([]NodeData, error) {
	farms, err := getFarms(client)
	if err != nil {
		return nil, err
	}
	nodes, err := client.AliveNodes()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't fetch nodes")
	}
	res := make([]NodeData, 0)
	for _, node := range nodes {
		if node.Status != "up" {
			continue
		}
		cap, err := freeCapacity(client, uint32(node.NodeID))
		if err != nil {
			return nil, err
		}
		farm, ok := farms[node.FarmID]
		if !ok {
			return nil, fmt.Errorf("farm %d not found", node.FarmID)
		}
		res = append(res, NodeData{
			ID:        uint32(node.NodeID),
			Farm:      farm,
			HasIPv4:   node.PublicConfig.Ipv4 != "",
			HasDomain: node.PublicConfig.Domain != "",
			Certified: true, // TODO: how to know
			Cap:       cap,
		})
	}
	return res, nil
}

func fullfils(node *NodeData, r *Request) bool {
	log.Printf("farm: %t\n", r.Farm != "" && r.Farm != node.Farm)
	log.Printf("cert: %t\n", r.Certified && !node.Certified)
	log.Printf("ipv4: %t\n", r.HasIPv4 && !node.HasIPv4)
	log.Printf("domain: %t\n", r.HasDomain && !node.HasDomain)
	log.Printf("cru: %t\n", r.Cap.CPUs > node.Cap.CPUs)
	log.Printf("hru: %t\n", r.Cap.Hru > node.Cap.Hru)
	log.Printf("sru: %t\n", r.Cap.Sru > node.Cap.Sru)
	log.Printf("memory: %t\n", r.Cap.Memory > node.Cap.Memory)
	if r.Farm != "" && r.Farm != node.Farm ||
		r.Certified && !node.Certified ||
		r.HasDomain && !node.HasDomain ||
		r.HasIPv4 && !node.HasIPv4 ||
		r.Cap.CPUs > node.Cap.CPUs ||
		r.Cap.Memory > node.Cap.Memory ||
		r.Cap.Hru > node.Cap.Hru ||
		r.Cap.Sru > node.Cap.Sru {
		return false
	}
	return true
}

func subtract(node *NodeData, r *Request) {
	node.Cap.CPUs -= r.Cap.CPUs
	node.Cap.Memory -= r.Cap.Memory
	node.Cap.Hru -= r.Cap.Hru
	node.Cap.Sru -= r.Cap.Sru
}

func schedule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	go startRmbIfNeeded(ctx, apiClient)
	nodes, err := getNodes(&apiClient.grid_client)
	if err != nil {
		return diag.FromErr(err)
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	assignmentIfs := d.Get("nodes").(map[string]interface{})
	assignment := make(map[string]uint32)
	for k, v := range assignmentIfs {
		assignment[k] = v.(uint32)
	}
	reqsIfs := d.Get("requests").([]interface{})
	reqs := make([]Request, 0)
	for _, r := range reqsIfs {
		mp := r.(map[string]interface{})
		if _, ok := assignment[mp["name"].(string)]; ok {
			// skip already assigned ones
			continue
		}
		reqs = append(reqs, Request{
			Name:      mp["name"].(string),
			Farm:      mp["farm"].(string),
			HasIPv4:   mp["ipv4"].(bool),
			HasDomain: mp["domain"].(bool),
			Certified: mp["certified"].(bool),
			Cap: MachineCapacity{
				CPUs:   uint64(mp["cru"].(int)),
				Memory: uint64(mp["mru"].(int)) * uint64(gridtypes.Megabyte),
				Hru:    uint64(mp["hru"].(int)) * uint64(gridtypes.Megabyte),
				Sru:    uint64(mp["sru"].(int)) * uint64(gridtypes.Megabyte),
			},
		})
	}
	log.Printf("requests length: %d\n", len(reqsIfs))
	log.Printf("nodes length: %d\n", len(nodes))
	curNode := 0
	for _, r := range reqs {
		i := 0
		for i < len(nodes) {
			json.NewEncoder(log.Writer()).Encode(nodes[curNode])
			if fullfils(&nodes[curNode], &r) {
				subtract(&nodes[curNode], &r)
				assignment[r.Name] = nodes[curNode].ID
				curNode = (curNode + 1) % len(nodes)
				break
			}
			curNode = (curNode + 1) % len(nodes)
			i++
		}
		if _, ok := assignment[r.Name]; !ok {
			json.NewEncoder(log.Writer()).Encode(r)
			return diag.FromErr(fmt.Errorf("didn't find a suitable node"))
		}
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
