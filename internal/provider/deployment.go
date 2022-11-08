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
	Id          string
	Node        uint32
	Disks       []workloads.Disk
	ZDBs        []workloads.ZDB
	VMs         []workloads.VM
	QSFSs       []workloads.QSFS
	IPRange     string
	NetworkName string
	APIClient   *apiClient
	ncPool      client.NodeClientCollection
	deployer    deployer.Deployer
}
type DeploymentData struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
}

func getDeploymentDeployer(d *schema.ResourceData, apiClient *apiClient) (DeploymentDeployer, error) {
	networkName := d.Get("network_name").(string)

	disks := make([]workloads.Disk, 0)
	for _, disk := range d.Get("disks").([]interface{}) {
		data := workloads.GetDiskData(disk.(map[string]interface{}))
		disks = append(disks, data)
	}

	zdbs := make([]workloads.ZDB, 0)
	for _, zdb := range d.Get("zdbs").([]interface{}) {
		data := workloads.GetZdbData(zdb.(map[string]interface{}))
		zdbs = append(zdbs, data)
	}

	vms := make([]workloads.VM, 0)
	for _, vm := range d.Get("vms").([]interface{}) {
		data := workloads.NewVMFromSchema(vm.(map[string]interface{})).WithNetworkName(networkName)
		vms = append(vms, *data)
	}

	qsfs := make([]workloads.QSFS, 0)
	for _, q := range d.Get("qsfs").([]interface{}) {
		data := workloads.NewQSFSFromSchema(q.(map[string]interface{}))
		qsfs = append(qsfs, data)
	}
	pool := client.NewNodeClientPool(apiClient.rmb)
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
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}

	deploymentDeployer := DeploymentDeployer{
		Id:          d.Id(),
		Node:        uint32(d.Get("node").(int)),
		Disks:       disks,
		VMs:         vms,
		QSFSs:       qsfs,
		ZDBs:        zdbs,
		IPRange:     d.Get("ip_range").(string),
		NetworkName: networkName,
		APIClient:   apiClient,
		ncPool:      pool,
		deployer:    deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, pool, true, solutionProvider, string(deploymentDataStr)),
	}
	return deploymentDeployer, nil
}

func (d *DeploymentDeployer) assignNodesIPs(sub subi.SubstrateExt) error {
	if len(d.VMs) == 0 {
		return nil
	}
	_, cidr, err := net.ParseCIDR(d.IPRange)
	if err != nil {
		return errors.Wrapf(err, "invalid ip %s", d.IPRange)
	}
	networkKey := deployer.GetNetworkKey(d.NetworkName)
	usedIPs, err := deployer.GetUsedIps(sub, d.APIClient.identity.PublicKey(), networkKey)
	// user defined ips
	for _, vm := range d.VMs {
		if vm.IP != "" && cidr.Contains(net.ParseIP(vm.IP)) {
			usedIPs = append(usedIPs, string(net.ParseIP(vm.IP)[3]))
		}
	}
	cur := byte(2)
	for idx, vm := range d.VMs {
		if vm.IP != "" && cidr.Contains(net.ParseIP(vm.IP)) {
			continue
		}
		ip := cidr.IP
		ip[3] = cur
		for isInStr(usedIPs, strconv.Itoa(int(ip[3]))) {
			if cur == 254 {
				return errors.New("all 253 ips of the network are exhausted")
			}
			cur++
			ip[3] = cur
		}
		d.VMs[idx].IP = ip.String()
		usedIPs = append(usedIPs, ip.String())
	}
	return nil
}
func (d *DeploymentDeployer) GenerateVersionlessDeployments(ctx context.Context, sub subi.SubstrateExt) (map[uint32]gridtypes.Deployment, error) {
	dl := workloads.NewDeployment(d.APIClient.twin_id)
	err := d.assignNodesIPs(sub)
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

	return map[uint32]gridtypes.Deployment{d.Node: dl}, nil
}

func (d *DeploymentDeployer) Marshal(r *schema.ResourceData) {
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
	r.Set("vms", vms)
	r.Set("zdbs", zdbs)
	r.Set("disks", disks)
	r.Set("qsfs", qsfs)
	r.Set("node", d.Node)
	r.Set("network_name", d.NetworkName)
	r.SetId(d.Id)
}

func (d *DeploymentDeployer) GetOldDeployments(ctx context.Context) (map[uint32]uint64, error) {
	deployments := make(map[uint32]uint64)
	if d.Id != "" {

		deploymentID, err := strconv.ParseUint(d.Id, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse deployment id %s", d.Id)
		}
		deployments[d.Node] = deploymentID
	}

	return deployments, nil
}
func (d *DeploymentDeployer) Nullify() {
	d.VMs = nil
	d.QSFSs = nil
	d.Disks = nil
	d.ZDBs = nil
	d.Id = ""
}
func (d *DeploymentDeployer) ID() uint64 {
	id, err := strconv.ParseUint(d.Id, 10, 64)
	if err != nil {
		panic(err)
	}
	return id

}
func (d *DeploymentDeployer) syncContract(sub subi.SubstrateExt) error {
	if d.Id == "" {
		return nil
	}
	valid, err := sub.IsValidContract(d.ID())
	if err != nil {
		return errors.Wrap(err, "error checking contract validity")
	}
	if !valid {
		d.Id = ""
		return nil
	}
	return nil
}
func (d *DeploymentDeployer) sync(ctx context.Context, sub subi.SubstrateExt) error {
	if err := d.syncContract(sub); err != nil {
		return err
	}
	if d.Id == "" {
		d.Nullify()
		return nil
	}
	currentDeployments, err := d.deployer.GetDeploymentObjects(ctx, sub, map[uint32]uint64{d.Node: d.ID()})
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}
	dl := currentDeployments[d.Node]
	var vms []workloads.VM
	var zdbs []workloads.ZDB
	var qsfs []workloads.QSFS
	var disks []workloads.Disk

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
func (d *DeploymentDeployer) validate() error {
	if len(d.VMs) != 0 && d.IPRange == "" {
		return errors.New("empty ip_range was passed," +
			" you probably used the wrong node id in the expression `lookup(grid_network.net1.nodes_ip_range, 4, \"\")`" +
			" the node id in the lookup must match the node property of the resource.")
	}
	if len(d.VMs) != 0 && strings.TrimSpace(d.IPRange) != d.IPRange {
		return errors.New("ip_range must not contain trailing or leading spaces")
	}
	_, _, err := net.ParseCIDR(d.IPRange)
	if len(d.VMs) != 0 && err != nil {
		return errors.Wrap(err, "If you pass a vm, ip_range must be set to a valid ip range (e.g. 10.1.3.0/16)")
	}
	if len(d.VMs) != 0 && d.NetworkName == "" {
		return errors.New("If you pass a vm, network_name must be non-empty")
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
	newDeployments, err := d.GenerateVersionlessDeployments(ctx, sub)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	oldDeployments, err := d.GetOldDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get old deployments data")
	}
	currentDeployments, err := d.deployer.Deploy(ctx, sub, oldDeployments, newDeployments)
	if currentDeployments[d.Node] != 0 {
		d.Id = fmt.Sprintf("%d", currentDeployments[d.Node])
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
	id := currentDeployments[d.Node]
	if id != 0 {
		d.Id = fmt.Sprintf("%d", id)
	} else {
		d.Id = ""
	}
	return err
}
