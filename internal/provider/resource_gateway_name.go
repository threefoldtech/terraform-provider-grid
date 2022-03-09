package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func resourceGatewayNameProxy() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource for deploying gateway domains.",

		CreateContext: resourceGatewayNameCreate,
		ReadContext:   resourceGatewayNameRead,
		UpdateContext: resourceGatewayNameUpdate,
		DeleteContext: resourceGatewayNameDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Gateway name (the fqdn will be <name>.<gateway-domain>)",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The gateway's node id",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The computed fully quallified domain name of the deployed workload.",
			},
			"tls_passthrough": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "True to pass the tls as is to the backends.",
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
			"name_contract_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The id of the name contract",
			},
		},
	}
}

type GatewayNameDeployer struct {
	Name           string
	Description    string
	Node           uint32
	TLSPassthrough bool
	Backends       []zos.Backend

	FQDN             string
	NodeDeploymentID map[uint32]uint64
	NameContractID   uint64

	APIClient *apiClient
	ncPool    *NodeClientPool
}

func NewGatewayNameDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (GatewayNameDeployer, error) {
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
			return GatewayNameDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	deployer := GatewayNameDeployer{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		Backends:         backends,
		FQDN:             d.Get("fqdn").(string),
		TLSPassthrough:   d.Get("tls_passthrough").(bool),
		NodeDeploymentID: nodeDeploymentID,
		APIClient:        apiClient,
		ncPool:           NewNodeClient(apiClient.rmb),
		NameContractID:   uint64(d.Get("name_contract_id").(int)),
	}
	return deployer, nil
}

func (k *GatewayNameDeployer) Validate(ctx context.Context, sub *substrate.Substrate) error {
	if err := validateAccountMoneyForExtrinsics(sub, k.APIClient.identity); err != nil {
		return err
	}
	return isNodesUp(ctx, sub, []uint32{k.Node}, k.ncPool)
}

func (k *GatewayNameDeployer) storeState(d *schema.ResourceData) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	d.Set("node", k.Node)
	d.Set("tls_passthrough", k.TLSPassthrough)
	d.Set("backends", k.Backends)
	d.Set("fqdn", k.FQDN)
	d.Set("node_deployment_id", nodeDeploymentID)
	d.Set("name_contract_id", k.NameContractID)
}
func (k *GatewayNameDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	workload := gridtypes.Workload{
		Version:     0,
		Type:        zos.GatewayNameProxyType,
		Description: k.Description,
		Name:        gridtypes.Name(k.Name),
		Data: gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name:           k.Name,
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

func (k *GatewayNameDeployer) ensureNameContract(ctx context.Context, sub *substrate.Substrate, name string) (uint64, error) {
	contractID, err := sub.GetContractIDByNameRegistration(name)
	if errors.Is(err, substrate.ErrNotFound) {
		if k.NameContractID != 0 { // the name changed, remove the old one
			if err := sub.CancelContract(k.APIClient.identity, k.NameContractID); err != nil {
				return 0, errors.Wrap(err, "couldn't delete the old name contract")
			}
		}
		contractID, err := sub.CreateNameContract(k.APIClient.identity, name)
		return contractID, errors.Wrap(err, "failed to create name contract")
	} else if err != nil {
		return 0, errors.Wrapf(err, "couldn't get the owning contract id of the name %s", name)
	}
	if contractID == k.NameContractID {
		return contractID, nil
	}
	contract, err := sub.GetContract(contractID)
	if err != nil {
		return 0, errors.Wrapf(err, "couldn't get the owning contract of the name %s", name)
	}
	if contract.TwinID != types.U32(k.APIClient.twin_id) {
		return 0, errors.Wrapf(err, "name already registered by twin id %d with contract id %d", contract.TwinID, contractID)
	}
	return contractID, nil
}

func (k *GatewayNameDeployer) Deploy(ctx context.Context, sub *substrate.Substrate) error {
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	cid, err := k.ensureNameContract(ctx, sub, k.Name)
	if err != nil {
		return err
	}
	k.NameContractID = cid
	currentDeployments, err := deployDeployments(ctx, sub, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, true)
	if err := k.updateState(ctx, sub, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}
func (k *GatewayNameDeployer) updateState(ctx context.Context, sub *substrate.Substrate, currentDeploymentIDs map[uint32]uint64) error {
	k.NodeDeploymentID = currentDeploymentIDs
	dls, err := getDeploymentObjects(ctx, sub, currentDeploymentIDs, k.ncPool)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl, ok := dls[k.Node]
	if !ok || len(dl.Workloads) == 0 {
		k.FQDN = ""
	} else {
		var result zos.GatewayProxyResult
		if err := json.Unmarshal(dl.Workloads[0].Result.Data, &result); err != nil {
			return errors.Wrap(err, "error unmarshalling json")
		}
		k.FQDN = result.FQDN
	}
	return nil
}

func (k *GatewayNameDeployer) updateFromRemote(ctx context.Context, sub *substrate.Substrate) error {
	return k.updateState(ctx, sub, k.NodeDeploymentID)
}

func (k *GatewayNameDeployer) Cancel(ctx context.Context, sub *substrate.Substrate) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	currentDeployments, err := deployDeployments(ctx, sub, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, false)
	// update even in case of error, then return the error after
	if err := k.updateState(ctx, sub, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	if err != nil {
		return err
	}
	if k.NameContractID != 0 {
		return sub.CancelContract(k.APIClient.identity, k.NameContractID)
	}
	return nil
}

func resourceGatewayNameCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	sub, err := apiClient.manager.Substrate()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
	}
	defer sub.Close()
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayNameDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := deployer.Validate(ctx, sub); err != nil {
		return diag.FromErr(err)
	}
	err = deployer.Deploy(ctx, sub)
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

func resourceGatewayNameUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	sub, err := apiClient.manager.Substrate()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
	}
	defer sub.Close()
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayNameDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, sub); err != nil {
		return diag.FromErr(err)
	}

	err = deployer.Deploy(ctx, sub)
	if err != nil {
		diags = diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

// TODO: make this non failing
func resourceGatewayNameRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	sub, err := apiClient.manager.Substrate()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
	}
	defer sub.Close()
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayNameDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	err = deployer.updateFromRemote(ctx, sub)
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

func resourceGatewayNameDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	sub, err := apiClient.manager.Substrate()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get substrate client"))
	}
	defer sub.Close()
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewGatewayNameDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	err = deployer.Cancel(ctx, sub)
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
