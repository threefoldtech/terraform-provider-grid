// Package provider is the terraform provider
package provider

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func newDeploymentFromSchema(d *schema.ResourceData) (*workloads.Deployment, error) {
	networkName := d.Get("network_name").(string)
	nodeID := uint32(d.Get("node").(int))
	name := d.Get("name").(string)
	solutionType := d.Get("solution_type").(string)

	disks := make([]workloads.Disk, 0)
	for _, disk := range d.Get("disks").([]interface{}) {
		d, err := workloads.NewWorkloadFromMap(disk.(map[string]interface{}), &workloads.Disk{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create workload from disk map")
		}
		disks = append(disks, *(d.(*workloads.Disk)))
	}

	zdbs := make([]workloads.ZDB, 0)
	for _, zdb := range d.Get("zdbs").([]interface{}) {
		z, err := workloads.NewWorkloadFromMap(zdb.(map[string]interface{}), &workloads.ZDB{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create workload from zdb map")
		}
		zdbs = append(zdbs, *(z.(*workloads.ZDB)))
	}

	vms := make([]workloads.VM, 0)
	for _, vm := range d.Get("vms").([]interface{}) {
		vmMap := vm.(map[string]interface{})
		vmMap["network_name"] = networkName
		v, err := workloads.NewWorkloadFromMap(vmMap, &workloads.VM{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create workload from vm map")
		}
		vms = append(vms, *v.(*workloads.VM))
	}

	// TODO: ip_range
	// err = r.Set("ip_range", d.IPRange)
	// if err != nil {
	// 	errors = multierror.Append(errors, err)
	// }

	qsfs := make([]workloads.QSFS, 0)
	for _, qsfsdata := range d.Get("qsfs").([]interface{}) {
		qsfsI := qsfsdata.(map[string]interface{})
		qsfsI["metadata"] = qsfsI["metadata"].([]interface{})[0]
		q, err := workloads.NewWorkloadFromMap(qsfsdata.(map[string]interface{}), &workloads.QSFS{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create workload from qsfs map")
		}
		qsfs = append(qsfs, *q.(*workloads.QSFS))
	}

	solutionProviderVal := uint64(d.Get("solution_provider").(int))
	var solutionProvider *uint64
	if solutionProviderVal == 0 {
		solutionProvider = nil
	} else {
		solutionProvider = &solutionProviderVal
	}

	var contractID uint64
	nodeDeploymentID := map[uint32]uint64{}
	var err error
	if d.Id() != "" {
		contractID, err = strconv.ParseUint(d.Id(), 10, 64)
		if err != nil {
			return nil, err
		}
		nodeDeploymentID[nodeID] = contractID
	}

	dl := workloads.Deployment{
		Name:             name,
		NodeID:           nodeID,
		SolutionProvider: solutionProvider,
		SolutionType:     solutionType,
		Disks:            disks,
		Vms:              vms,
		QSFS:             qsfs,
		Zdbs:             zdbs,
		NetworkName:      networkName,
		ContractID:       contractID,
		NodeDeploymentID: nodeDeploymentID,
	}

	return &dl, nil
}

// syncContractsDeployments updates the terraform local state with the latest changes to workloads
func syncContractsDeployments(r *schema.ResourceData, d *workloads.Deployment) (errors error) {
	vms := make([]interface{}, 0)
	disks := make([]interface{}, 0)
	zdbs := make([]interface{}, 0)
	qsfs := make([]interface{}, 0)
	for _, vm := range d.Vms {
		vmMap, err := workloads.ToMap(vm)
		if err != nil {
			return err
		}
		delete(vmMap, "network_name")
		vms = append(vms, vmMap)
	}
	for _, d := range d.Disks {
		disk, err := workloads.ToMap(d)
		if err != nil {
			return err
		}
		disks = append(disks, disk)
	}
	for _, z := range d.Zdbs {
		zdb, err := workloads.ToMap(z)
		if err != nil {
			return err
		}
		zdbs = append(zdbs, zdb)
	}
	for _, q := range d.QSFS {
		qs, err := workloads.ToMap(q)
		if err != nil {
			return err
		}

		qs["metadata"] = []interface{}{qs["metadata"]}
		qsfs = append(qsfs, qs)
	}

	err := r.Set("vms", vms)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set vms with error: %w", err))
	}

	err = r.Set("zdbs", zdbs)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set zdbs with error: %w", err))
	}

	err = r.Set("disks", disks)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set disks with error: %w", err))
	}

	err = r.Set("qsfs", qsfs)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set qsfs with error: %w", err))
	}

	err = r.Set("node", d.NodeID)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set node with error: %w", err))
	}

	err = r.Set("network_name", d.NetworkName)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set network name with error: %w", err))
	}

	err = r.Set("solution_type", d.SolutionType)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set solution type with error: %w", err))
	}

	var solutionProvider int
	if d.SolutionProvider != nil {
		solutionProvider = int(*d.SolutionProvider)
	}
	err = r.Set("solution_provider", solutionProvider)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set solution provider with error: %w", err))
	}

	/* TODO: iprange
	err = r.Set("ip_range", d.IPRange)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to set ip range with error: %w", err))
	}
	*/

	r.SetId(fmt.Sprint(d.ContractID))
	return
}
