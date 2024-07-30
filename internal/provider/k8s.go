// Package provider is the terraform provider
package provider

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// newK8sFromSchema reads the k8s resource configuration data from the schema.ResourceData, converts them into a new K8s instance, and returns this instance.
func newK8sFromSchema(d *schema.ResourceData) (*workloads.K8sCluster, error) {
	nodesIPRange := make(map[uint32]gridtypes.IPNet)

	masterMap := d.Get("master").([]interface{})[0].(map[string]interface{})

	myceliumIPSeed := masterMap["mycelium_ip_seed"].(string)
	myceliumIPSeedBytes, err := hex.DecodeString(myceliumIPSeed)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode mycelium ip seed '%s'", myceliumIPSeed)
	}
	masterMap["mycelium_ip_seed"] = myceliumIPSeedBytes

	masterI, err := workloads.NewWorkloadFromMap(masterMap, &workloads.K8sNode{})
	if err != nil {
		return nil, err
	}

	workers := make([]workloads.K8sNode, 0)

	for _, w := range d.Get("workers").([]interface{}) {
		wMap := w.(map[string]interface{})

		myceliumIPSeed := wMap["mycelium_ip_seed"].(string)
		myceliumIPSeedBytes, err := hex.DecodeString(myceliumIPSeed)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode mycelium ip seed '%s'", myceliumIPSeed)
		}
		wMap["mycelium_ip_seed"] = myceliumIPSeedBytes

		data, err := workloads.NewWorkloadFromMap(wMap, &workloads.K8sNode{})
		if err != nil {
			return nil, err
		}
		workers = append(workers, *data.(*workloads.K8sNode))
	}

	nodeDeploymentIDIf := d.Get("node_deployment_id").(map[string]interface{})
	nodeDeploymentID := make(map[uint32]uint64)
	for node, id := range nodeDeploymentIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse node id %s'", node)
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}
	master := masterI.(*workloads.K8sNode)
	solutionType := d.Get("solution_type").(string)

	if solutionType == "" {
		solutionType = fmt.Sprintf("kubernetes/%s", master.Name)
	}
	k8s := workloads.K8sCluster{
		Master:           master,
		Workers:          workers,
		Token:            d.Get("token").(string),
		SSHKey:           d.Get("ssh_key").(string),
		NetworkName:      d.Get("network_name").(string),
		SolutionType:     solutionType,
		NodeDeploymentID: nodeDeploymentID,
		NodesIPRange:     nodesIPRange,
	}
	return &k8s, nil
}

func retainChecksums(workers []interface{}, master interface{}, k8s *workloads.K8sCluster) {
	checksumMap := make(map[string]string)
	checksumMap[k8s.Master.Name] = k8s.Master.FlistChecksum
	for _, w := range k8s.Workers {
		checksumMap[w.Name] = w.FlistChecksum
	}
	typed := master.(map[string]interface{})
	typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	for _, w := range workers {
		typed := w.(map[string]interface{})
		typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	}
}

func storeK8sState(d *schema.ResourceData, k8s *workloads.K8sCluster) (errors error) {
	workers := make([]interface{}, 0)
	for _, w := range k8s.Workers {
		wMap, err := workloads.ToMap(w)
		if err != nil {
			return err
		}
		wMap["mycelium_ip_seed"] = hex.EncodeToString(k8s.Master.MyceliumIPSeed)
		workers = append(workers, wMap)
	}

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k8s.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	nodeIPRanges := make(map[string]interface{})
	for node, ip := range k8s.NodesIPRange {
		nodeIPRanges[fmt.Sprintf("%d", node)] = ip.String()
	}

	if k8s.Master == nil {
		k8s.Master = &workloads.K8sNode{}
	}

	master, err := workloads.ToMap(k8s.Master)
	if err != nil {
		return err
	}

	master["mycelium_ip_seed"] = hex.EncodeToString(k8s.Master.MyceliumIPSeed)
	retainChecksums(workers, master, k8s)

	l := []interface{}{master}
	err = d.Set("master", l)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("workers", workers)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("token", k8s.Token)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("ssh_key", k8s.SSHKey)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("network_name", k8s.NetworkName)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("solution_type", k8s.SolutionType)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("nodes_ip_range", nodeIPRanges)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	return
}
