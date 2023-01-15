// Package provider is the terraform provider
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type GatewayNameDeployer struct {
	Gw workloads.GatewayNameProxy

	ID               string
	Node             uint32
	Description      string
	NodeDeploymentID map[uint32]uint64
	NameContractID   uint64

	ThreefoldPluginClient *threefoldPluginClient
	ncPool                client.NodeClientGetter
	deployer              deployer.Deployer
}

func NewGatewayNameDeployer(d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (GatewayNameDeployer, error) {
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
	pool := client.NewNodeClientPool(threefoldPluginClient.rmb)
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
		ID:               d.ID(),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		NodeDeploymentID: nodeDeploymentID,
		NameContractID:   uint64(d.Get("name_contract_id").(int)),

		ThreefoldPluginClient: threefoldPluginClient,
		ncPool:                pool,
		deployer:              deployer.NewDeployer(threefoldPluginClient.identity, threefoldPluginClient.twinID, threefoldPluginClient.gridProxyClient, pool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

func (k *GatewayNameDeployer) Validate(ctx context.Context, sub subi.SubstrateExt) error {
	return client.AreNodesUp(ctx, sub, []uint32{k.Node}, k.ncPool)
}

func (k *GatewayNameDeployer) SyncContractsDeployments(d *schema.ResourceData) (errors error) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	d.SetId(k.ID)
	err := d.Set("node", k.Node)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("tls_passthrough", k.Gw.TLSPassthrough)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("backends", k.Gw.Backends)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("fqdn", k.Gw.FQDN)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("name_contract_id", k.NameContractID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	return
}

func (k *GatewayNameDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	deployment := workloads.NewDeployment(k.ThreefoldPluginClient.twinID)
	deployment.Workloads = append(deployment.Workloads, k.Gw.ZosWorkload())
	deployments[k.Node] = deployment
	return deployments, nil
}
func (k *GatewayNameDeployer) InvalidateNameContract(ctx context.Context, sub subi.SubstrateExt) (err error) {
	if k.NameContractID == 0 {
		return
	}

	k.NameContractID, err = sub.InvalidateNameContract(
		ctx,
		k.ThreefoldPluginClient.identity,
		k.NameContractID,
		k.Gw.Name,
	)
	return
}
func (k *GatewayNameDeployer) Deploy(ctx context.Context, sub subi.SubstrateExt) error {
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
		k.NameContractID, err = sub.CreateNameContract(k.ThreefoldPluginClient.identity, k.Gw.Name)
		if err != nil {
			return err
		}
	}
	if k.ID == "" {
		// create the resource if the contract is created
		k.ID = uuid.New().String()
	}
	k.NodeDeploymentID, err = k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	return err
}
func (k *GatewayNameDeployer) syncContracts(ctx context.Context, sub subi.SubstrateExt) (err error) {
	if err := sub.DeleteInvalidContracts(k.NodeDeploymentID); err != nil {
		return err
	}
	valid, err := sub.IsValidContract(k.NameContractID)
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

// Sync syncs the deployments
func (k *GatewayNameDeployer) Sync(ctx context.Context, sub subi.SubstrateExt, cl *threefoldPluginClient) (err error) {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}
	dls, err := k.deployer.GetDeployments(ctx, sub, k.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl := dls[k.Node]
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

func (k *GatewayNameDeployer) Cancel(ctx context.Context, sub subi.SubstrateExt) (err error) {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	k.NodeDeploymentID, err = k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	if err != nil {
		return err
	}
	if k.NameContractID != 0 {
		if err := sub.EnsureContractCanceled(k.ThreefoldPluginClient.identity, k.NameContractID); err != nil {
			return err
		}
		k.NameContractID = 0
	}
	return nil
}
