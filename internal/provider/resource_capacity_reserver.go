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
	Farm               uint32 `name:"farm"`
	CapacityContractID uint64 `name:"capacity_contract_id"`
	CPU                int    `name:"cpu"`
	Memory             int    `name:"memory"`
	SSD                int    `name:"ssd"`
	HDD                int    `name:"hdd"`
	SolutionProvider   uint64 `name:"solution_provider"`
	Public             bool   `name:"public"`
	Node               uint32 `name:"node"`
	GroupID            uint32 `name:"group_id"`
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
			"capacity_contract_id": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of nodes to add to the network",
			},
			"farm": {
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
			"node": {
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
	_, err := apiClient.substrateConn.GetContract(capacityReserver.CapacityContractID)
	if err != nil && errors.Is(err, substrate.ErrNotFound) {
		d.SetId("")
	} else if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
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

	d.SetId(strconv.FormatUint(capacityReserver.CapacityContractID, 10))
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
		Farm:               d.Get("farm").(uint32),
		CapacityContractID: d.Get("capacity_contract_id").(uint64),
		CPU:                d.Get("cpu").(int),
		Memory:             d.Get("memory").(int),
		SSD:                d.Get("ssd").(int),
		HDD:                d.Get("hdd").(int),
		SolutionProvider:   d.Get("solution_provider").(uint64),
		Public:             d.Get("public").(bool),
		Node:               d.Get("node").(uint32),
		GroupID:            d.Get("group_id").(uint32),
	}
}

func (c *Capacity) updateState(d *schema.ResourceData) error {
	var setErr error
	err := d.Set("cpu", c.CPU)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("capacity_contract_id", c.CapacityContractID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("farm", c.Farm)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("group_id", c.GroupID)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("hdd", c.HDD)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("memory", c.Memory)
	setErr = errors.Wrap(setErr, err.Error())
	err = d.Set("node", c.Node)
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
	contractID, err := cl.substrateConn.CreateCapacityReservationContract(cl.identity, c.Farm, policy, solutionProvider)
	if err != nil {
		return err
	}
	contract, err := cl.substrateConn.GetContract(contractID)
	if err != nil {
		return err
	}
	c.Node = uint32(contract.ContractType.CapacityReservationContract.NodeID)
	c.CapacityContractID = contractID
	return nil
}

func (c *Capacity) Update(cl apiClient) error {
	resource := substrate.Resources{
		HRU: types.U64(c.HDD),
		SRU: types.U64(c.SSD),
		MRU: types.U64(c.Memory),
		CRU: types.U64(c.CPU),
	}
	err := cl.substrateConn.UpdateCapacityReservationContract(cl.identity, c.CapacityContractID, resource)
	if err != nil {
		return err
	}
	return nil
}

func (c *Capacity) Delete(cl apiClient) error {
	return cl.substrateConn.CancelContract(cl.identity, c.CapacityContractID)
}

// checks if a non-changeable field has changed, and warn user if so
func validateCapacityChanges(d *schema.ResourceData) error {
	nonChangeable := []string{"farm", "solution_provider", "public", "group_id"}
	for _, field := range nonChangeable {
		if d.HasChange(field) {
			return ErrNonChangeable
		}
	}
	return nil
}
