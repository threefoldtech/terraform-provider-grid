// Package provider is the terraform provider
package provider

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// newNameGatewayFromSchema reads the gateway_name_proxy resource configuration data from schema.ResourceData, converts them into a GatewayName instance, then returns this instance.
func newNameGatewayFromSchema(d *schema.ResourceData) (*workloads.GatewayNameProxy, error) {
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
			return nil, errors.Wrapf(err, "couldn't parse node id '%v'", node)
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	var contractID uint64
	var err error
	if d.Id() != "" {
		contractID, err = strconv.ParseUint(d.Id(), 10, 64)
		if err != nil {
			return nil, err
		}
	}

	gw := workloads.GatewayNameProxy{
		NodeID:           uint32(d.Get("node").(int)),
		Name:             d.Get("name").(string),
		Backends:         backends,
		TLSPassthrough:   d.Get("tls_passthrough").(bool),
		Description:      d.Get("description").(string),
		SolutionType:     d.Get("solution_type").(string),
		Network:          d.Get("network").(string),
		FQDN:             d.Get("fqdn").(string),
		NodeDeploymentID: nodeDeploymentID,
		NameContractID:   uint64(d.Get("name_contract_id").(int)),
		ContractID:       contractID,
	}
	return &gw, nil
}

// syncContractsNameGateways updates the terraform local state with the resource's latest changes.
func syncContractsNameGateways(d *schema.ResourceData, gw *workloads.GatewayNameProxy) (errors error) {
	nodeDeploymentID := make(map[string]interface{})
	for node, id := range gw.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	err := d.Set("node", gw.NodeID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("tls_passthrough", gw.TLSPassthrough)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("backends", gw.Backends)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("network", gw.Network)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("fqdn", gw.FQDN)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("name_contract_id", gw.NameContractID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	d.SetId(fmt.Sprint(gw.ContractID))
	return
}
