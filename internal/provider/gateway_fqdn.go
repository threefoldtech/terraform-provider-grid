package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type GatewayFQDNDeployer struct {
	Gw                            workloads.GatewayFQDNProxy
	ID                            string
	Description                   string
	Node                          uint32
	CapacityReservationContractID uint64
	ContractDeploymentID          map[uint64]uint64

	APIClient *apiClient
	ncPool    client.NodeClientCollection
	deployer  deployer.Deployer
}

func NewGatewayFQDNDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (GatewayFQDNDeployer, error) {
	backendsIf := d.Get("backends").([]interface{})
	backends := make([]zos.Backend, len(backendsIf))
	for idx, n := range backendsIf {
		backends[idx] = zos.Backend(n.(string))
	}
	capacityReservationContractID := d.Get("capacity_reservation_contract_id").(uint64)
	ContractDeploymentIDIf := d.Get("contract_deployment_id").(map[string]interface{})
	ContractDeploymentID := make(map[uint64]uint64)
	for contractID, deploymentID := range ContractDeploymentIDIf {
		contractIDInt, err := strconv.ParseUint(contractID, 10, 64)
		if err != nil {
			return GatewayFQDNDeployer{}, errors.Wrap(err, "couldn't parse contract id")
		}
		deploymentIDInt := uint64(deploymentID.(int))
		ContractDeploymentID[contractIDInt] = deploymentIDInt
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
	deployer := GatewayFQDNDeployer{
		Gw: workloads.GatewayFQDNProxy{
			Name:           d.Get("name").(string),
			Backends:       backends,
			FQDN:           d.Get("fqdn").(string),
			TLSPassthrough: d.Get("tls_passthrough").(bool),
		},
		ID:                            d.Id(),
		Description:                   d.Get("description").(string),
		Node:                          uint32(d.Get("node").(int)),
		ContractDeploymentID:          ContractDeploymentID,
		CapacityReservationContractID: capacityReservationContractID,
		APIClient:                     apiClient,
		ncPool:                        ncPool,
		deployer:                      deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, ncPool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

func (k *GatewayFQDNDeployer) Validate(ctx context.Context, sub *substrate.Substrate) error {
	contract, err := sub.GetContract(k.CapacityReservationContractID)
	if err != nil {
		return errors.Wrapf(err, "couldn't get contract %d info", k.CapacityReservationContractID)
	}
	k.Node = uint32(contract.ContractType.CapacityReservationContract.NodeID)
	return isNodesUp(ctx, sub, []uint32{k.Node}, k.ncPool)
}

func (k *GatewayFQDNDeployer) Marshal(d *schema.ResourceData) error {

	contractDeploymentID := make(map[string]interface{})
	for contractID, deploymentID := range k.ContractDeploymentID {
		contractDeploymentID[fmt.Sprintf("%d", contractID)] = int(deploymentID)
	}

	err := errSet{}
	err.Push(d.Set("node", k.Node))
	err.Push(d.Set("tls_passthrough", k.Gw.TLSPassthrough))
	err.Push(d.Set("backends", k.Gw.Backends))
	err.Push(d.Set("fqdn", k.Gw.FQDN))
	err.Push(d.Set("contract_deployment_id", contractDeploymentID))
	d.SetId(k.ID)
	return err.error()
}
func (k *GatewayFQDNDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint64]gridtypes.Deployment, error) {
	deployments := make(map[uint64]gridtypes.Deployment)
	dl := workloads.NewDeployment(k.APIClient.twin_id)
	dl.Workloads = append(dl.Workloads, k.Gw.ZosWorkload())
	deployments[k.CapacityReservationContractID] = dl
	return deployments, nil
}

func (k *GatewayFQDNDeployer) Deploy(ctx context.Context, sub *substrate.Substrate) error {
	if err := k.Validate(ctx, sub); err != nil {
		return err
	}
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	k.ContractDeploymentID, err = k.deployer.Deploy(ctx, sub, k.ContractDeploymentID, newDeployments)
	if k.ID == "" && k.ContractDeploymentID[k.CapacityReservationContractID] != 0 {
		k.ID = strconv.FormatUint(k.ContractDeploymentID[k.CapacityReservationContractID], 10)
	}
	return err
}

func (k *GatewayFQDNDeployer) syncContracts(ctx context.Context, sub *substrate.Substrate) (err error) {
	if err := DeleteInvalidContracts(sub, k.ContractDeploymentID); err != nil {
		return err
	}
	if len(k.ContractDeploymentID) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		k.ID = ""
	}
	return nil
}
func (k *GatewayFQDNDeployer) sync(ctx context.Context, sub *substrate.Substrate, cl *apiClient) error {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}

	dls, err := k.deployer.GetDeploymentObjects(ctx, sub, k.ContractDeploymentID)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl := dls[k.CapacityReservationContractID]
	wl, _ := dl.Get(gridtypes.Name(k.Gw.Name))
	k.Gw = workloads.GatewayFQDNProxy{}
	if wl != nil && wl.Result.State.IsOkay() {
		k.Gw, err = workloads.GatewayFQDNProxyFromZosWorkload(*wl.Workload)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *GatewayFQDNDeployer) Cancel(ctx context.Context, sub *substrate.Substrate) (err error) {
	newDeployments := make(map[uint64]gridtypes.Deployment)

	k.ContractDeploymentID, err = k.deployer.Deploy(ctx, sub, k.ContractDeploymentID, newDeployments)

	return err
}
