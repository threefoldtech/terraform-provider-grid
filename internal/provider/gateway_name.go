package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type GatewayNameDeployer struct {
	Gw               workloads.GatewayNameProxy
	ID               string
	Description      string
	Node             uint32
	NodeDeploymentID map[uint32]uint64
	DeploymentData   deployer.DeploymentData
	DeploymentProps  deployer.DeploymentProps
	NameContractID   uint64

	DeployerClient *deployer.Client
	APIClient      *apiClient
	ncPool         client.NodeClientCollection
	deployer       deployer.SingleDeployerInterface
}

func NewGatewayNameDeployer(d *schema.ResourceData, apiClient *apiClient) (GatewayNameDeployer, error) {
	backendsIf := d.Get("backends").([]interface{})
	backends := make([]zos.Backend, len(backendsIf))
	for idx, n := range backendsIf {
		backends[idx] = zos.Backend(n.(string))
	}
	capacityReservationContractID := d.Get("capacity_reservation_contract_id").(uint64)
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
	ncPool := client.NewNodeClientPool(apiClient.rmb)
	deploymentData := DeploymentData{
		Name:        d.Get("name").(string),
		Type:        "gateway",
		ProjectName: d.Get("solution_type").(string),
	}
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}
	gw := workloads.GatewayNameProxy{
		Name:           d.Get("name").(string),
		Backends:       backends,
		FQDN:           d.Get("fqdn").(string),
		TLSPassthrough: d.Get("tls_passthrough").(bool),
	}
	dl := workloads.NewDeployment(apiClient.twin_id)
	dl.Workloads = append(dl.Workloads, gw.ZosWorkload())
	deploymentProps := deployer.DeploymentProps{
		Deployment: dl,
		ContractID: deployer.CapacityReservationContractID(capacityReservationContractID),
	}
	deployerClinet := &deployer.Client{
		Identity:  apiClient.identity,
		Sub:       apiClient.substrateConn,
		Twin:      apiClient.twin_id,
		NCPool:    ncPool,
		GridProxy: apiClient.grid_client,
	}
	deployer := GatewayNameDeployer{
		Gw:               gw,
		ID:               d.Id(),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		NodeDeploymentID: nodeDeploymentID,
		DeploymentData:   deployer.DeploymentData(deploymentDataStr),
		DeploymentProps:  deploymentProps,
		NameContractID:   uint64(d.Get("name_contract_id").(int)),
		DeployerClient:   deployerClinet,
		APIClient:        apiClient,
		ncPool:           ncPool,
		deployer:         &deployer.SingleDeployer{},
	}
	return deployer, nil
}

func (k *GatewayNameDeployer) Marshal(d *schema.ResourceData) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	d.SetId(k.ID)
	d.Set("node", k.Node)
	d.Set("tls_passthrough", k.Gw.TLSPassthrough)
	d.Set("backends", k.Gw.Backends)
	d.Set("fqdn", k.Gw.FQDN)
	d.Set("node_deployment_id", nodeDeploymentID)
	d.Set("name_contract_id", k.NameContractID)
}

func (k *GatewayNameDeployer) InvalidateNameContract(ctx context.Context, sub *substrate.Substrate) (err error) {
	if k.NameContractID == 0 {
		return
	}

	k.NameContractID, err = InvalidateNameContract(
		sub,
		k.APIClient.identity,
		k.NameContractID,
		k.Gw.Name,
	)
	return
}
func (k *GatewayNameDeployer) Create(ctx context.Context, sub *substrate.Substrate) error {
	err := k.InvalidateNameContract(ctx, sub)
	if err != nil {
		return err
	}
	if k.NameContractID == 0 {
		k.NameContractID, err = sub.CreateNameContract(k.APIClient.identity, k.Gw.Name)
		if err != nil {
			return err
		}
	}
	if k.ID == "" {
		// create the resource if the contract is created
		k.ID = uuid.New().String()
	}
	err = k.deployer.Create(
		ctx,
		k.DeployerClient,
		k.DeploymentData,
		&k.DeploymentProps,
	)
	return err
}
func (k *GatewayNameDeployer) Update(ctx context.Context, sub *substrate.Substrate) error {
	err := k.InvalidateNameContract(ctx, sub)
	if err != nil {
		return err
	}
	if k.NameContractID == 0 {
		k.NameContractID, err = sub.CreateNameContract(k.APIClient.identity, k.Gw.Name)
		if err != nil {
			return err
		}
	}
	err = k.deployer.Update(
		ctx,
		k.DeployerClient,
		k.DeploymentData,
		&k.DeploymentProps,
	)
	return err
}
func (k *GatewayNameDeployer) syncContracts(ctx context.Context, sub *substrate.Substrate) (err error) {
	if err := DeleteInvalidContracts(sub, k.NodeDeploymentID); err != nil {
		return err
	}
	valid, err := IsValidContract(sub, k.NameContractID)
	if err != nil {
		return err
	}
	if !valid {
		k.NameContractID = 0
	}
	if k.NameContractID == 0 && len(k.NodeDeploymentID) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		k.ID = ""
	}
	return nil
}
func (k *GatewayNameDeployer) sync(ctx context.Context, sub *substrate.Substrate, cl *apiClient) (err error) {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}
	dl, err := k.deployer.GetCurrentState(
		ctx,
		k.DeployerClient,
		&k.DeploymentProps,
	)
	wl, _ := dl.Get(gridtypes.Name(k.Gw.Name))
	k.Gw = workloads.GatewayNameProxy{}
	// if the node acknowledges it, we are golden
	if wl != nil && wl.Result.State.IsOkay() {
		k.Gw, err = workloads.GatewayNameProxyFromZosWorkload(*wl.Workload)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *GatewayNameDeployer) Delete(ctx context.Context, sub *substrate.Substrate) (err error) {
	err = k.deployer.Delete(
		ctx,
		k.DeployerClient,
		deployer.DeploymentID(k.NodeDeploymentID[k.Node]),
	)
	if err != nil {
		return err
	}
	if k.NameContractID != 0 {
		if err := EnsureContractCanceled(sub, k.APIClient.identity, k.NameContractID); err != nil {
			return err
		}
		k.NameContractID = 0
	}
	return nil
}
