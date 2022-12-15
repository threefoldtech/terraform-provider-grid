package provider

import (
	"context"
	"strconv"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

type Capacity struct {
	FarmID           uint32 `name:"farm_id"`
	CapacityID       uint64 `name:"capacity_id"`
	CPU              int    `name:"cpu"`
	Memory           int    `name:"memory"`
	SSD              int    `name:"ssd"`
	HDD              int    `name:"hdd"`
	SolutionProvider uint64 `name:"solution_provider"`
	Public           bool   `name:"public"`
	NodeID           uint32 `name:"node_id"`
	GroupID          uint32 `name:"group_id"`
}

var ErrNonChangeable = errors.New("this field cannot be updated. please delete the resource and recreate it")

func resourceCapacityReserver() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for computing deployments requirements.",

		ReadContext:   resourceCapacityRead,
		CreateContext: resourceCapacityCreate,
		UpdateContext: resourceCapacityUpdate,
		DeleteContext: resourceCapacityDelete,

		Schema: map[string]*schema.Schema{
			"capacity_id": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of nodes to add to the network",
			},
			"farm_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Farm id of deployment",
			},
			"solution_provider": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"cpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Number of VCPUs",
			},
			"memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Memory size",
			},
			"ssd": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "SSD size",
			},
			"hdd": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "HDD size",
			},
			"public": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "node has public ip",
			},
			"node_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "node id",
			},
			"group_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "group id that this contract should belong to",
			},
		},
	}
}

func resourceCapacityRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(apiClient)
	var diags diag.Diagnostics
	capacityReserver := NewCapacity(d)
	_, err := apiClient.substrateConn.GetContract(capacityReserver.CapacityID)
	if err != nil {
		switch {
		case errors.Is(err, substrate.ErrNotFound):
			d.SetId("")
		default:
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}
	return nil
}
func resourceCapacityCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(apiClient)
	var diags diag.Diagnostics
	capacityReserver := NewCapacity(d)

	err := capacityReserver.Create(apiClient)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// set state
	err = capacityReserver.updateState(d)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	// this will call reserve capacity and set node ids in deployments

	d.SetId(strconv.FormatUint(capacityReserver.CapacityID, 10))
	return diags
}
func resourceCapacityUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(apiClient)
	var diags diag.Diagnostics
	err := validateCapacityChanges(d)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	capacityReserver := NewCapacity(d)

	err = capacityReserver.Update(apiClient)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	err = capacityReserver.updateState(d)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	return diags
}
func resourceCapacityDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(apiClient)
	var diags diag.Diagnostics
	capacityReserver := NewCapacity(d)

	err := capacityReserver.Delete(apiClient)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	d.SetId("")
	return nil
}

func NewCapacity(d *schema.ResourceData) Capacity {
	return Capacity{
		FarmID:           d.Get("farm_id").(uint32),
		CapacityID:       d.Get("capacity_id").(uint64),
		CPU:              d.Get("cpu").(int),
		Memory:           d.Get("memory").(int),
		SSD:              d.Get("ssd").(int),
		HDD:              d.Get("hdd").(int),
		SolutionProvider: d.Get("solution_provider").(uint64),
		Public:           d.Get("public").(bool),
		NodeID:           d.Get("node_id").(uint32),
		GroupID:          d.Get("group_id").(uint32),
	}
}

func (c *Capacity) updateState(d *schema.ResourceData) error {
	var setErr error
	err := d.Set("cpu", c.CPU)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("capacity_id", c.CapacityID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("farm_id", c.FarmID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("group_id", c.GroupID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("hdd", c.HDD)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("memory", c.Memory)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("node_id", c.NodeID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("public", c.Public)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("ssd", c.SSD)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("solution_provider", c.SolutionProvider)
	setErr = errors.Wrap(setErr, err.Error())
	return setErr
}

func (c *Capacity) Create(cl apiClient) error {
	resource := substrate.Resources{
		HRU: types.U64(c.HDD),
		SRU: types.U64(c.SSD),
		MRU: types.U64(c.Memory),
		CRU: types.U64(c.CPU),
	}
	feature := substrate.NodeFeatures{
		IsPublicNode: c.Public,
	}
	var policy substrate.CapacityReservationPolicy
	if c.GroupID != 0 {
		policy = substrate.WithExclusivePolicy(c.GroupID, resource, feature)
	} else {
		policy = substrate.WithCapacityPolicy(resource, feature)
	}
	var solutionProvider *uint64
	if c.SolutionProvider != 0 {
		solutionProvider = &c.SolutionProvider
	} else {
		solutionProvider = nil
	}
	capacityID, err := cl.substrateConn.CreateCapacityReservationContract(cl.identity, c.FarmID, policy, solutionProvider)
	if err != nil {
		return err
	}
	contract, err := cl.substrateConn.GetContract(capacityID)
	if err != nil {
		return err
	}
	c.NodeID = uint32(contract.ContractType.CapacityReservationContract.NodeID)
	c.CapacityID = capacityID
	return nil
}

func (c *Capacity) Update(cl apiClient) error {
	resource := substrate.Resources{
		HRU: types.U64(c.HDD),
		SRU: types.U64(c.SSD),
		MRU: types.U64(c.Memory),
		CRU: types.U64(c.CPU),
	}
	return cl.substrateConn.UpdateCapacityReservationContract(cl.identity, c.CapacityID, resource)

}

func (c *Capacity) Delete(cl apiClient) error {
	return cl.substrateConn.CancelContract(cl.identity, c.CapacityID)
}

// checks if a non-changeable field has changed, and warn user if so
func validateCapacityChanges(d *schema.ResourceData) error {
	nonChangeable := []string{"farm_id", "solution_provider", "public", "group_id"}
	for _, field := range nonChangeable {
		if d.HasChange(field) {
			return ErrNonChangeable
		}
	}
	return nil
}
