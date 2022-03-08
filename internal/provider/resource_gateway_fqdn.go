package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func resourceGatewayFQDNProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying gateway domains.",

		CreateContext: resourceGatewayFQDNCreate,
		ReadContext:   resourceGatewayFQDNRead,
		UpdateContext: resourceGatewayFQDNUpdate,
		DeleteContext: resourceGatewayFQDNDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "name",
				Description: "Gateway workload name (of no actual significance)",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description field",
			},
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The gateway's node id",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The fully quallified domain name of the deployed workload",
			},
			"tls_passthrough": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "true to pass the tls as is to the backends",
			},
			"backends": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The backends of the gateway proxy (in the format (http|https)://ip:port), with tls_passthrough the scheme must be https",
			},
			"node_deployment_id": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each node to its deployment id",
			},
		},
	}
}

type GatewayFQDNDeployer struct {
	Name           string
	Description    string
	Node           uint32
	TLSPassthrough bool
	Backends       []zos.Backend

	FQDN             string
	NodeDeploymentID map[uint32]uint64

	APIClient *apiClient
	ncPool    *NodeClientPool
}

func NewGatewayFQDNDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (GatewayFQDNDeployer, error) {
	backendsIf := d.Get("backends").([]interface{})
	backends := make([]zos.Backend, len(backendsIf))
	for idx, n := range backendsIf {
		backends[idx] = zos.Backend(n.(string))
	}
	nodeDeploymentIDIf := d.Get("node_deployment_id").(map[string]interface{})
	nodeDeploymentID := make(map[uint32]uint64)
	for node, id := range nodeDeploymentIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return GatewayFQDNDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	deployer := GatewayFQDNDeployer{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		Backends:         backends,
		FQDN:             d.Get("fqdn").(string),
		TLSPassthrough:   d.Get("tls_passthrough").(bool),
		NodeDeploymentID: nodeDeploymentID,
		APIClient:        apiClient,
		ncPool:           NewNodeClient(apiClient.manager, apiClient.rmb),
	}
	return deployer, nil
}

func (k *GatewayFQDNDeployer) Validate(ctx context.Context) error {
	if err := validateAccountMoneyForExtrinsics(k.APIClient); err != nil {
		return err
	}
	return isNodesUp(ctx, []uint32{k.Node}, k.ncPool)
}

func (k *GatewayFQDNDeployer) ValidateRead(ctx context.Context) error {
	nodes := make([]uint32, 0)
	for node := range k.NodeDeploymentID {
		nodes = append(nodes, node)
	}
	return isNodesUp(ctx, nodes, k.ncPool)
}

func (k *GatewayFQDNDeployer) ValidateDelete(ctx context.Context) error {
	return nil
}

func (k *GatewayFQDNDeployer) storeState(d *schema.ResourceData) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	d.Set("node", k.Node)
	d.Set("tls_passthrough", k.TLSPassthrough)
	d.Set("backends", k.Backends)
	d.Set("fqdn", k.FQDN)
	d.Set("node_deployment_id", nodeDeploymentID)
}
func (k *GatewayFQDNDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	workload := gridtypes.Workload{
		Version:     0,
		Type:        zos.GatewayFQDNProxyType,
		Description: k.Description,
		Name:        gridtypes.Name(k.Name),
		Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN:           k.FQDN,
			TLSPassthrough: k.TLSPassthrough,
			Backends:       k.Backends,
		}),
	}

	deployment := gridtypes.Deployment{
		Version: Version,
		TwinID:  k.APIClient.twin_id, //LocalTwin,
		// this contract id must match the one on substrate
		Workloads: []gridtypes.Workload{
			workload,
		},
		SignatureRequirement: gridtypes.SignatureRequirement{
			WeightRequired: 1,
			Requests: []gridtypes.SignatureRequest{
				{
					TwinID: k.APIClient.twin_id,
					Weight: 1,
				},
			},
		},
	}
	deployments[k.Node] = deployment
	return deployments, nil
}

func (k *GatewayFQDNDeployer) GetOldDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	return getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
}

func (k *GatewayFQDNDeployer) Deploy(ctx context.Context) error {
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	currentDeployments, err := deployDeployments(ctx, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, true)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}
func (k *GatewayFQDNDeployer) updateState(ctx context.Context, currentDeploymentIDs map[uint32]uint64) error {
	k.NodeDeploymentID = currentDeploymentIDs
	dls, err := getDeploymentObjects(ctx, currentDeploymentIDs, k.ncPool)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl, ok := dls[k.Node]
	if !ok {
		k.FQDN = ""
	} else {
		data, err := dl.Workloads[0].WorkloadData()
		if err != nil {
			return errors.Wrap(err, "error getting workload data")
		}
		k.FQDN = data.(*zos.GatewayFQDNProxy).FQDN
	}
	return nil
}

func (k *GatewayFQDNDeployer) updateFromRemote(ctx context.Context) error {
	return k.updateState(ctx, k.NodeDeploymentID)
}

func (k *GatewayFQDNDeployer) Cancel(ctx context.Context) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)

	currentDeployments, err := deployDeployments(ctx, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, false)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func resourceGatewayFQDNCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := deployer.Validate(ctx); err != nil {
		return diag.FromErr(err)
	}
	err = deployer.Deploy(ctx)
	if err != nil {
		if len(deployer.NodeDeploymentID) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	deployer.storeState(d)
	d.SetId(uuid.New().String())
	return diags
}

func resourceGatewayFQDNUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx); err != nil {
		return diag.FromErr(err)
	}

	err = deployer.Deploy(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func resourceGatewayFQDNRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	err = deployer.updateFromRemote(ctx)
	log.Printf("read updateFromRemote err: %s\n", err)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   err.Error(),
		})
		return diags
	}
	deployer.storeState(d)
	return diags
}

func resourceGatewayFQDNDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayFQDNDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	err = deployer.Cancel(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
	} else {
		deployer.storeState(d)
	}
	return diags
}
