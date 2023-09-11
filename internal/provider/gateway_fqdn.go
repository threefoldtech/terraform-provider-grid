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

// newFQDNGatewayFromSchema reads the gateway_fqdn_proxy resource configuration data from schema.ResourceData, converts them into a GatewayFQDND instance, then returns this instance.
func newFQDNGatewayFromSchema(d *schema.ResourceData) (*workloads.GatewayFQDNProxy, error) {
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

	tlsPassthrough := d.Get("tls_passthrough").(bool)
	if err := validateBackends(backends, tlsPassthrough); err != nil {
		return nil, err
	}

	gw := workloads.GatewayFQDNProxy{
		NodeID:           uint32(d.Get("node").(int)),
		Name:             d.Get("name").(string),
		Backends:         backends,
		FQDN:             d.Get("fqdn").(string),
		TLSPassthrough:   tlsPassthrough,
		Network:          d.Get("network").(string),
		SolutionType:     d.Get("solution_type").(string),
		Description:      d.Get("description").(string),
		NodeDeploymentID: nodeDeploymentID,
		ContractID:       contractID,
	}
	return &gw, nil
}

// syncContractsFQDNGateways updates the terraform local state with the resource's latest changes.
func syncContractsFQDNGateways(d *schema.ResourceData, gw *workloads.GatewayFQDNProxy) (errors error) {
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

	err = d.Set("fqdn", gw.FQDN)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("network", gw.Network)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	d.SetId(fmt.Sprint(gw.ContractID))
	return
}
