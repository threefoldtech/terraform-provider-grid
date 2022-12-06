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
	Gw               workloads.GatewayFQDNProxy
	ID               string
	Description      string
	Node             uint32
	NodeDeploymentID map[uint32]uint64
	DeploymentData   deployer.DeploymentData
	DeploymentProps  deployer.DeploymentProps

	DeployerClient *deployer.Client
	APIClient      *apiClient
	ncPool         client.NodeClientCollection
	deployer       deployer.SingleDeployerInterface
}

func NewGatewayFQDNDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (GatewayFQDNDeployer, error) {
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
			return GatewayFQDNDeployer{}, errors.Wrap(err, "couldn't parse node id")
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
	gw := workloads.GatewayFQDNProxy{
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
	deployer := GatewayFQDNDeployer{
		Gw:               gw,
		ID:               d.Id(),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		NodeDeploymentID: nodeDeploymentID,
		DeploymentData:   deployer.DeploymentData(deploymentDataStr),
		DeploymentProps:  deploymentProps,
		DeployerClient:   deployerClinet,
		APIClient:        apiClient,
		ncPool:           ncPool,
		deployer:         &deployer.SingleDeployer{},
	}
	return deployer, nil
}

func (k *GatewayFQDNDeployer) Marshal(d *schema.ResourceData) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	d.Set("node", k.Node)
	d.Set("tls_passthrough", k.Gw.TLSPassthrough)
	d.Set("backends", k.Gw.Backends)
	d.Set("fqdn", k.Gw.FQDN)
	d.Set("node_deployment_id", nodeDeploymentID)
	d.SetId(k.ID)
}

func (k *GatewayFQDNDeployer) Create(ctx context.Context, sub *substrate.Substrate) error {
	err := k.deployer.Create(
		ctx,
		k.DeployerClient,
		k.DeploymentData,
		&k.DeploymentProps,
	)
	if err == nil {
		k.NodeDeploymentID[k.Node] = k.DeploymentProps.Deployment.DeploymentID.U64()
	}
	if k.ID == "" && k.NodeDeploymentID[k.Node] != 0 {
		k.ID = strconv.FormatUint(k.NodeDeploymentID[k.Node], 10)
	}
	return err
}

func (k *GatewayFQDNDeployer) Update(ctx context.Context, sub *substrate.Substrate) error {
	err := k.deployer.Update(
		ctx,
		k.DeployerClient,
		k.DeploymentData,
		&k.DeploymentProps,
	)
	return err
}

func (k *GatewayFQDNDeployer) syncContracts(ctx context.Context, sub *substrate.Substrate) (err error) {
	if err := DeleteInvalidContracts(sub, k.NodeDeploymentID); err != nil {
		return err
	}
	if len(k.NodeDeploymentID) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		k.ID = ""
	}
	return nil
}
func (k *GatewayFQDNDeployer) sync(ctx context.Context, sub *substrate.Substrate, cl *apiClient) error {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}

	dl, err := k.deployer.GetCurrentState(
		ctx,
		k.DeployerClient,
		&k.DeploymentProps,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
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

func (k *GatewayFQDNDeployer) Delete(ctx context.Context, sub *substrate.Substrate) (err error) {
	err = k.deployer.Delete(
		ctx,
		k.DeployerClient,
		deployer.DeploymentID(k.NodeDeploymentID[k.Node]),
	)

	return err
}
