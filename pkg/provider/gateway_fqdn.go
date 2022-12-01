package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	client "github.com/threefoldtech/terraform-provider-grid/pkg/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
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
	deployer := GatewayFQDNDeployer{
		Gw: workloads.GatewayFQDNProxy{
			Name:           d.Get("name").(string),
			Backends:       backends,
			FQDN:           d.Get("fqdn").(string),
			TLSPassthrough: d.Get("tls_passthrough").(bool),
		},
		ID:               d.Id(),
		Description:      d.Get("description").(string),
		Node:             uint32(d.Get("node").(int)),
		NodeDeploymentID: nodeDeploymentID,
		APIClient:        apiClient,
		ncPool:           ncPool,
		deployer:         deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, ncPool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

func (k *GatewayFQDNDeployer) Validate(ctx context.Context, sub subi.SubstrateExt) error {
	return isNodesUp(ctx, sub, []uint32{k.Node}, k.ncPool)
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
func (k *GatewayFQDNDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	dl := workloads.NewDeployment(k.APIClient.twin_id)
	dl.Workloads = append(dl.Workloads, k.Gw.ZosWorkload())
	deployments[k.Node] = dl
	return deployments, nil
}

func (k *GatewayFQDNDeployer) Deploy(ctx context.Context, sub subi.SubstrateExt) error {
	if err := k.Validate(ctx, sub); err != nil {
		return err
	}
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	k.NodeDeploymentID, err = k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	if k.ID == "" && k.NodeDeploymentID[k.Node] != 0 {
		k.ID = strconv.FormatUint(k.NodeDeploymentID[k.Node], 10)
	}
	return err
}

func (k *GatewayFQDNDeployer) syncContracts(ctx context.Context, sub subi.SubstrateExt) (err error) {
	if err := sub.DeleteInvalidContracts(k.NodeDeploymentID); err != nil {
		return err
	}
	if len(k.NodeDeploymentID) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		k.ID = ""
	}
	return nil
}
func (k *GatewayFQDNDeployer) sync(ctx context.Context, sub subi.SubstrateExt, cl *apiClient) error {
	if err := k.syncContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't sync contracts")
	}

	dls, err := k.deployer.GetDeploymentObjects(ctx, sub, k.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment objects")
	}
	dl := dls[k.Node]
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

func (k *GatewayFQDNDeployer) Cancel(ctx context.Context, sub subi.SubstrateExt) (err error) {
	newDeployments := make(map[uint32]gridtypes.Deployment)

	k.NodeDeploymentID, err = k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)

	return err
}
