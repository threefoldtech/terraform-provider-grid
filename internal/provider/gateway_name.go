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
	Gw workloads.GatewayNameProxy

	ID                    string
	Node                  uint32
	Description           string
	NameContractID        uint64
	CapacityID            uint64
	CapacityDeploymentMap map[uint64]uint64

	APIClient *apiClient
	ncPool    client.NodeClientCollection
	deployer  deployer.Deployer
}

func NewGatewayNameDeployer(d *schema.ResourceData, apiClient *apiClient) (GatewayNameDeployer, error) {
	backendsIf := d.Get("backends").([]interface{})
	backends := make([]zos.Backend, len(backendsIf))
	for idx, n := range backendsIf {
		backends[idx] = zos.Backend(n.(string))
	}
	capacityID := d.Get("capacity_id").(uint64)
	contractDeploymentMapIf := d.Get("capacity_deployment_map").(map[string]interface{})
	capacityDeploymentMap := make(map[uint64]uint64)
	for contractID, deploymentID := range contractDeploymentMapIf {
		contractIDInt, err := strconv.ParseUint(contractID, 10, 64)
		if err != nil {
			return GatewayNameDeployer{}, errors.Wrap(err, "couldn't parse contract id")
		}
		deploymentIDInt := uint64(deploymentID.(int))
		capacityDeploymentMap[contractIDInt] = deploymentIDInt
	}
	pool := client.NewNodeClientPool(apiClient.rmb)
	deploymentData := DeploymentData{
		Name:        d.Get("name").(string),
		Type:        "gateway",
		ProjectName: d.Get("solution_type").(string),
	}
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}
	deployer := GatewayNameDeployer{
		Gw: workloads.GatewayNameProxy{
			Name:           d.Get("name").(string),
			Backends:       backends,
			FQDN:           d.Get("fqdn").(string),
			TLSPassthrough: d.Get("tls_passthrough").(bool),
		},
		ID:                    d.Id(),
		Description:           d.Get("description").(string),
		Node:                  uint32(d.Get("node").(int)),
		CapacityDeploymentMap: capacityDeploymentMap,
		CapacityID:            capacityID,
		NameContractID:        uint64(d.Get("name_contract_id").(int)),

		APIClient: apiClient,
		ncPool:    pool,
		deployer:  deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, pool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

func (k *GatewayNameDeployer) Validate(ctx context.Context, sub *substrate.Substrate) error {
	contract, err := sub.GetContract(k.CapacityID)
	if err != nil {
		return errors.Wrapf(err, "couldn't get contract %d info", k.CapacityID)
	}
	k.Node = uint32(contract.ContractType.CapacityReservationContract.NodeID)
	return isNodesUp(ctx, sub, []uint32{k.Node}, k.ncPool)
}

func (k *GatewayNameDeployer) Marshal(d *schema.ResourceData) error {

	capacityDeploymentMap := make(map[string]interface{})
	for contractID, deploymentID := range k.CapacityDeploymentMap {
		capacityDeploymentMap[fmt.Sprintf("%d", contractID)] = int(deploymentID)
	}

	err := errSet{}
	d.SetId(k.ID)

	err.Push(d.Set("node", k.Node))
	err.Push(d.Set("tls_passthrough", k.Gw.TLSPassthrough))
	err.Push(d.Set("backends", k.Gw.Backends))
	err.Push(d.Set("fqdn", k.Gw.FQDN))
	err.Push(d.Set("capacity_deployment_map", capacityDeploymentMap))
	err.Push(d.Set("name_contract_id", k.NameContractID))
	return err.error()
}

func (k *GatewayNameDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint64]gridtypes.Deployment, error) {
	deployments := make(map[uint64]gridtypes.Deployment)
	deployment := workloads.NewDeployment(k.APIClient.twin_id)
	deployment.Workloads = append(deployment.Workloads, k.Gw.ZosWorkload())
	deployments[k.CapacityID] = deployment
	return deployments, nil
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
func (k *GatewayNameDeployer) Deploy(ctx context.Context, sub *substrate.Substrate) error {
	if err := k.Validate(ctx, sub); err != nil {
		return err
	}
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	if err := k.InvalidateNameContract(ctx, sub); err != nil {
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
	k.CapacityDeploymentMap, err = k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
	return err
}
func (k *GatewayNameDeployer) syncContracts(ctx context.Context, sub *substrate.Substrate) (err error) {
	if err := DeleteInvalidContracts(sub, k.CapacityDeploymentMap); err != nil {
		return err
	}
	valid, err := IsValidContract(sub, k.NameContractID)
	if err != nil {
		return err
	}
	if !valid {
		k.NameContractID = 0
	}
	if k.NameContractID == 0 && len(k.CapacityDeploymentMap) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		k.ID = ""
	}
	return nil
}
func (k *GatewayNameDeployer) sync(ctx context.Context, sub *substrate.Substrate, cl *apiClient) (err error) {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}
	dls, err := k.deployer.GetDeploymentObjects(ctx, sub, k.CapacityDeploymentMap)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl := dls[k.CapacityID]
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

func (k *GatewayNameDeployer) Cancel(ctx context.Context, sub *substrate.Substrate) (err error) {
	newDeployments := make(map[uint64]gridtypes.Deployment)
	k.CapacityDeploymentMap, err = k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
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
