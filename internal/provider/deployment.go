// Package provider is the terraform provider
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"

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

type DeploymentDeployer struct {
	ID                    string
	Node                  uint32
	Disks                 []workloads.Disk
	ZDBs                  []workloads.ZDB
	VMs                   []workloads.VM
	QSFSs                 []workloads.QSFS
	IPRange               string
	NetworkName           string
	ThreefoldPluginClient *threefoldPluginClient
	ncPool                client.NodeClientGetter
	deployer              deployer.Deployer
}
type DeploymentData struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
}

func newDeploymentDeployer(d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (DeploymentDeployer, error) {
	networkName := d.Get("network_name").(string)
	nodeID := uint32(d.Get("node").(int))
	disks := make([]workloads.Disk, 0)
	for _, disk := range d.Get("disks").([]interface{}) {
		d := workloads.NewDiskFromSchema(disk.(map[string]interface{}))
		disks = append(disks, d)
	}

	zdbs := make([]workloads.ZDB, 0)
	for _, zdb := range d.Get("zdbs").([]interface{}) {
		z := workloads.NewZDBFromSchema(zdb.(map[string]interface{}))
		zdbs = append(zdbs, z)
	}

	vms := make([]workloads.VM, 0)
	for _, vm := range d.Get("vms").([]interface{}) {
		v := workloads.NewVMFromSchema(vm.(map[string]interface{})).WithNetworkName(networkName)
		vms = append(vms, *v)
	}

	qsfs := make([]workloads.QSFS, 0)
	for _, qsfsdata := range d.Get("qsfs").([]interface{}) {
		q := workloads.NewQSFSFromSchema(qsfsdata.(map[string]interface{}))
		qsfs = append(qsfs, q)
	}
	pool := client.NewNodeClientPool(threefoldPluginClient.rmb)
	solutionProviderVal := uint64(d.Get("solution_provider").(int))
	var solutionProvider *uint64
	if solutionProviderVal == 0 {
		solutionProvider = nil
	} else {
		solutionProvider = &solutionProviderVal
	}
	deploymentData := DeploymentData{
		Name:        d.Get("name").(string),
		Type:        "vm",
		ProjectName: d.Get("solution_type").(string),
	}
	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deployment data: %s", err.Error())
	}

	networkingState := threefoldPluginClient.state.GetState().Networks
	net := networkingState.GetNetwork(networkName)
	ipRange := net.GetNodeSubnet(nodeID)

	deploymentDeployer := DeploymentDeployer{
		ID:                    d.Id(),
		Node:                  nodeID,
		Disks:                 disks,
		VMs:                   vms,
		QSFSs:                 qsfs,
		ZDBs:                  zdbs,
		IPRange:               ipRange,
		NetworkName:           networkName,
		ThreefoldPluginClient: threefoldPluginClient,
		ncPool:                pool,
		deployer:              deployer.NewDeployer(threefoldPluginClient.identity, threefoldPluginClient.twinID, threefoldPluginClient.gridProxyClient, pool, true, solutionProvider, string(deploymentDataBytes)),
	}
	return deploymentDeployer, nil
}

func (d *DeploymentDeployer) assignNodesHostIDs() error {
	networkingState := d.ThreefoldPluginClient.state.GetState().Networks
	network := networkingState.GetNetwork(d.NetworkName)
	usedHosts := network.GetUsedNetworkHostIDs(d.Node)
	if len(d.VMs) == 0 {
		return nil
	}
	ip, ipRangeCIDR, err := net.ParseCIDR(d.IPRange)
	if err != nil {
		return errors.Wrapf(err, "invalid ip %s", d.IPRange)
	}
	for _, vm := range d.VMs {
		vmIP := net.ParseIP(vm.IP)
		vmHostID := vmIP[3]
		if vm.IP != "" && ipRangeCIDR.Contains(vmIP) && !Contains(usedHosts, vmHostID) {
			usedHosts = append(usedHosts, vmHostID)
		}
	}
	curHostID := byte(2)

	for idx, vm := range d.VMs {

		if vm.IP != "" && ipRangeCIDR.Contains(net.ParseIP(vm.IP)) {
			continue
		}

		for Contains(usedHosts, curHostID) {
			if curHostID == 254 {
				return errors.New("all 253 ips of the network are exhausted")
			}
			curHostID++
		}
		usedHosts = append(usedHosts, curHostID)
		vmIP := ip
		vmIP[3] = curHostID
		d.VMs[idx].IP = vmIP.String()
	}
	return nil
}
func (d *DeploymentDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	dl := workloads.NewDeployment(d.ThreefoldPluginClient.twinID)
	err := d.assignNodesHostIDs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	for _, disk := range d.Disks {
		dl.Workloads = append(dl.Workloads, disk.GenerateDiskWorkload())
	}
	for _, zdb := range d.ZDBs {
		dl.Workloads = append(dl.Workloads, zdb.GenerateZDBWorkload())
	}
	for _, vm := range d.VMs {
		dl.Workloads = append(dl.Workloads, vm.GenerateVMWorkload()...)
	}

	for idx, q := range d.QSFSs {
		qsfsWorkload, err := q.ZosWorkload()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate qsfs %d", idx)
		}
		dl.Workloads = append(dl.Workloads, qsfsWorkload)
	}

	return map[uint32]gridtypes.Deployment{d.NodeID: dl}, nil
}

func (d *DeploymentDeployer) SyncContractsDeployments(r *schema.ResourceData) (errors error) {
	vms := make([]interface{}, 0)
	disks := make([]interface{}, 0)
	zdbs := make([]interface{}, 0)
	qsfs := make([]interface{}, 0)
	for _, vm := range d.VMs {
		vms = append(vms, vm.Dictify())
	}
	for _, d := range d.Disks {
		disks = append(disks, d.Dictify())
	}
	for _, zdb := range d.ZDBs {
		zdbs = append(zdbs, zdb.Dictify())
	}
	for _, q := range d.QSFSs {
		qsfs = append(zdbs, q.Dictify())
	}
	err := r.Set("vms", vms)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("zdbs", zdbs)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("disks", disks)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("qsfs", qsfs)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("node", d.NodeID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("network_name", d.NetworkName)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = r.Set("ip_range", d.IPRange)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	r.SetId(d.ID)
	return
}

func (d *DeploymentDeployer) GetOldDeployments(ctx context.Context) (map[uint32]uint64, error) {
	deployments := make(map[uint32]uint64)
	if d.ID != "" {

		deploymentID, err := strconv.ParseUint(d.ID, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse deployment id %s", d.ID)
		}
		deployments[d.NodeID] = deploymentID
	}

	return deployments, nil
}
func (d *DeploymentDeployer) Nullify() {
	d.VMs = nil
	d.QSFSs = nil
	d.Disks = nil
	d.ZDBs = nil
	d.ID = ""
}
func (d *DeploymentDeployer) parseID() uint64 {
	id, err := strconv.ParseUint(d.ID, 10, 64)
	if err != nil {
		panic(err)
	}
	return id

}
func (d *DeploymentDeployer) syncContract(sub subi.SubstrateExt) error {
	if d.ID == "" {
		return nil
	}
	valid, err := sub.IsValidContract(d.parseID())
	if err != nil {
		return errors.Wrap(err, "error checking contract validity")
	}
	if !valid {
		d.ID = ""
		return nil
	}
	return nil
}

// Sync syncs the deployments
func (d *DeploymentDeployer) Sync(ctx context.Context, sub subi.SubstrateExt, cl *threefoldPluginClient) error {
	if err := d.syncContract(sub); err != nil {
		return err
	}
	if d.ID == "" {
		d.Nullify()
		return nil
	}
	currentDeployments, err := d.deployer.GetDeployments(ctx, sub, map[uint32]uint64{d.Node: d.parseID()})
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}
	dl := currentDeployments[d.NodeID]
	var vms []workloads.VM
	var zdbs []workloads.ZDB
	var qsfs []workloads.QSFS
	var disks []workloads.Disk

	ns := cl.state.GetState().Networks
	network := ns.GetNetwork(d.NetworkName)
	network.DeleteDeploymentHostIDs(d.Node, d.ID)

	usedIPs := []byte{}
	for _, w := range dl.Workloads {
		if !w.Result.State.IsOkay() {
			continue
		}
		switch w.Type {
		case zos.ZMachineType:
			vm, err := workloads.NewVMFromWorkloads(&w, &dl)
			if err != nil {
				log.Printf("error parsing vm: %s", err.Error())
				continue
			}
			vms = append(vms, vm)

			ip := net.ParseIP(vm.IP).To4()
			usedIPs = append(usedIPs, ip[3])
		case zos.ZDBType:
			zdb, err := workloads.NewZDBFromWorkload(&w)
			if err != nil {
				log.Printf("error parsing zdb: %s", err.Error())
				continue
			}
			zdbs = append(zdbs, zdb)
		case zos.QuantumSafeFSType:
			q, err := workloads.NewQSFSFromWorkload(&w)
			if err != nil {
				log.Printf("error parsing qsfs: %s", err.Error())
				continue
			}
			qsfs = append(qsfs, q)
		case zos.ZMountType:
			disk, err := workloads.NewDiskFromWorkload(&w)
			if err != nil {
				log.Printf("error parsing disk: %s", err.Error())
				continue
			}
			disks = append(disks, disk)

		}
	}
	network.SetDeploymentHostIDs(d.Node, d.ID, usedIPs)
	d.Match(disks, qsfs, zdbs, vms)
	log.Printf("vms: %+v\n", len(vms))
	d.Disks = disks
	d.QSFSs = qsfs
	d.ZDBs = zdbs
	d.VMs = vms
	return nil
}

// Match objects to match the input
//
//	already existing object are stored ordered the same way they are in the input
//	others are pushed after
func (d *DeploymentDeployer) Match(
	disks []workloads.Disk,
	qsfs []workloads.QSFS,
	zdbs []workloads.ZDB,
	vms []workloads.VM,
) {
	vmMap := make(map[string]*workloads.VM)
	l := len(d.Disks) + len(d.QSFSs) + len(d.ZDBs) + len(d.VMs)
	names := make(map[string]int)
	for idx, o := range d.Disks {
		names[o.Name] = idx - l
	}
	for idx, o := range d.QSFSs {
		names[o.Name] = idx - l
	}
	for idx, o := range d.ZDBs {
		names[o.Name] = idx - l
	}
	for idx, o := range d.VMs {
		names[o.Name] = idx - l
		vmMap[o.Name] = &d.VMs[idx]
	}
	sort.Slice(disks, func(i, j int) bool {
		return names[disks[i].Name] < names[disks[j].Name]
	})
	sort.Slice(qsfs, func(i, j int) bool {
		return names[qsfs[i].Name] < names[qsfs[j].Name]
	})
	sort.Slice(zdbs, func(i, j int) bool {
		return names[zdbs[i].Name] < names[zdbs[j].Name]
	})
	sort.Slice(vms, func(i, j int) bool {
		return names[vms[i].Name] < names[vms[j].Name]
	})
	for idx := range vms {
		vm, ok := vmMap[vms[idx].Name]
		if ok {
			vms[idx].Match(vm)
			log.Printf("orig: %+v\n", vm)
			log.Printf("new: %+v\n", vms[idx])
		}
	}
}

// TODO: are there any more validations on workloads needed other than vm and networkname relation?
func (d *DeploymentDeployer) validate() error {
	if len(d.VMs) != 0 && len(strings.TrimSpace(d.NetworkName)) == 0 {
		return errors.New("if you pass a vm, network_name must be non-empty")
	}

	for _, vm := range d.VMs {
		if err := vm.Validate(); err != nil {
			return errors.Wrapf(err, "vm %s validation failed", vm.Name)
		}
	}
	return nil
}
func (d *DeploymentDeployer) Deploy(ctx context.Context, sub subi.SubstrateExt) error {
	if err := d.validate(); err != nil {
		return err
	}
	newDeployments, err := d.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	oldDeployments, err := d.GetOldDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get old deployments data")
	}
	currentDeployments, err := d.deployer.Deploy(ctx, sub, oldDeployments, newDeployments)
	if currentDeployments[d.Node] != 0 {
		d.ID = fmt.Sprintf("%d", currentDeployments[d.Node])
	}
	return err
}

func (d *DeploymentDeployer) Cancel(ctx context.Context, sub subi.SubstrateExt) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	oldDeployments, err := d.GetOldDeployments(ctx)
	if err != nil {
		return err
	}
	currentDeployments, err := d.deployer.Deploy(ctx, sub, oldDeployments, newDeployments)
	id := currentDeployments[d.NodeID]
	if id != 0 {
		d.ID = fmt.Sprintf("%d", id)
	} else {
		d.ID = ""
	}
	return err
}
