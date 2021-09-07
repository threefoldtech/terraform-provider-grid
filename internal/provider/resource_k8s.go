package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func resourceKubernetes() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		CreateContext: resourceK8sCreate,
		ReadContext:   resourceK8sRead,
		UpdateContext: resourceK8sUpdate,
		DeleteContext: resourceK8sDelete,

		Schema: map[string]*schema.Schema{
			"node_deployment_id": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			"network_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"token": {
				Description: "The cluster secret token",
				Type:        schema.TypeString,
				Required:    true,
			},
			"nodes_ip_range": {
				Type:     schema.TypeMap,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"master": {
				MaxItems: 1,
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"node": {
							Description: "Node ID",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"disk_size": {
							Description: "Data disk size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"publicip": {
							Description: "If you want to enable public ip or not",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"flist": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "https://hub.grid.tf/ahmed_hanafy_1/ahmedhanafy725-k3s-latest.flist",
						},
						"computedip": {
							Description: "The public ip",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"ip": {
							Description: "IP",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"cpu": {
							Description: "CPU size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"memory": {
							Description: "Memory size",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"workers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"flist": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "https://hub.grid.tf/ahmed_hanafy_1/ahmedhanafy725-k3s-latest.flist",
						},
						"disk_size": {
							Description: "Data disk size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"node": {
							Description: "Node ID",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"publicip": {
							Description: "If you want to enable public ip or not",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"computedip": {
							Description: "The public ip",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"ip": {
							Description: "IP",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"cpu": {
							Description: "CPU size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"memory": {
							Description: "Memory size",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
		},
	}
}

type K8sNodeData struct {
	Name       string
	Node       uint32
	DiskSize   int
	PublicIP   bool
	Flist      string
	ComputedIP string
	IP         string
	Cpu        int
	Memory     int
}

type K8sDeployer struct {
	Master           *K8sNodeData
	Workers          []K8sNodeData
	NodesIPRange     map[uint32]gridtypes.IPNet
	Token            string
	SSHKey           string
	NetworkName      string
	NodeDeploymentID map[uint32]uint64

	APIClient *apiClient

	UsedIPs map[uint32][]string
	ncPool  *NodeClientPool
	d       *schema.ResourceData
}

func NewK8sNodeData(m map[string]interface{}) K8sNodeData {
	return K8sNodeData{
		Name:       m["name"].(string),
		Node:       uint32(m["node"].(int)),
		DiskSize:   m["disk_size"].(int),
		PublicIP:   m["publicip"].(bool),
		Flist:      m["flist"].(string),
		ComputedIP: m["computedip"].(string),
		IP:         m["ip"].(string),
		Cpu:        m["cpu"].(int),
		Memory:     m["memory"].(int),
	}
}

func NewK8sNodeDataFromWorkload(w gridtypes.Workload, nodeID uint32, diskSize int, computedIP string) (K8sNodeData, error) {
	var k K8sNodeData
	data, err := w.WorkloadData()
	if err != nil {
		return k, err
	}
	d := data.(*zos.ZMachine)
	k = K8sNodeData{
		Name:       string(w.Name),
		Node:       nodeID,
		DiskSize:   diskSize,
		PublicIP:   !d.Network.PublicIP.IsEmpty(),
		Flist:      d.FList,
		ComputedIP: computedIP,
		IP:         d.Network.Interfaces[0].IP.String(),
		Cpu:        int(d.ComputeCapacity.CPU),
		Memory:     int(d.ComputeCapacity.Memory / gridtypes.Megabyte),
	}
	return k, nil
}

func NewK8sDeployer(d *schema.ResourceData, apiClient *apiClient) (K8sDeployer, error) {
	master := NewK8sNodeData(d.Get("master").([]interface{})[0].(map[string]interface{}))
	workers := make([]K8sNodeData, 0)
	usedIPs := make(map[uint32][]string)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRangeIf := d.Get("nodes_ip_range").(map[string]interface{})
	for node, r := range nodesIPRangeIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return K8sDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		nodesIPRange[uint32(nodeInt)], err = gridtypes.ParseIPNet(r.(string))
		if err != nil {
			return K8sDeployer{}, errors.Wrap(err, "couldn't parse node ip range")
		}
	}
	if master.IP != "" {
		usedIPs[master.Node] = append(usedIPs[master.Node], master.IP)
	}
	for _, w := range d.Get("workers").([]interface{}) {
		data := NewK8sNodeData(w.(map[string]interface{}))
		workers = append(workers, data)
		if data.IP != "" {
			usedIPs[data.Node] = append(usedIPs[data.Node], data.IP)
		}
	}
	nodeDeploymentIDIf := d.Get("node_deployment_id").(map[string]interface{})
	nodeDeploymentID := make(map[uint32]uint64)
	for node, id := range nodeDeploymentIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return K8sDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	deployer := K8sDeployer{
		Master:           &master,
		Workers:          workers,
		Token:            d.Get("token").(string),
		SSHKey:           d.Get("ssh_key").(string),
		NetworkName:      d.Get("network_name").(string),
		NodeDeploymentID: nodeDeploymentID,
		UsedIPs:          usedIPs,
		NodesIPRange:     nodesIPRange,
		APIClient:        apiClient,
		ncPool:           NewNodeClient(apiClient.sub, apiClient.rmb),
		d:                d,
	}
	return deployer, nil
}

func (k *K8sNodeData) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = k.Name
	res["node"] = int(k.Node)
	res["disk_size"] = k.DiskSize
	res["publicip"] = k.PublicIP
	res["flist"] = k.Flist
	res["computedip"] = k.ComputedIP
	res["ip"] = k.IP
	res["cpu"] = k.Cpu
	res["memory"] = k.Memory
	return res
}

func (k *K8sDeployer) storeState(d *schema.ResourceData) {
	workers := make([]interface{}, 0)
	for _, w := range k.Workers {
		workers = append(workers, w.Dictify())
	}
	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}
	log.Printf("master data: %v\n", k.Master)
	if k.Master == nil {
		l := make([]interface{}, 0)
		x := K8sNodeData{}
		l = append(l, x.Dictify())
		d.Set("master", l)
	} else {
		l := make([]interface{}, 0)
		l = append(l, k.Master.Dictify())
		d.Set("master", l)
	}
	d.Set("workers", workers)
	d.Set("token", k.Token)
	d.Set("ssh_key", k.SSHKey)
	d.Set("network_name", k.NetworkName)
	d.Set("node_deployment_id", nodeDeploymentID)
}

func (k *K8sDeployer) assignNodesIPs() error {
	// TODO: when a k8s node changes its zos node, remove its ip from the used ones. better at the beginning
	masterNodeRange := k.NodesIPRange[k.Master.Node]
	if k.Master.IP == "" || !masterNodeRange.Contains(net.ParseIP(k.Master.IP)) {
		ip, err := getK8sFreeIP(masterNodeRange, k.UsedIPs[k.Master.Node])
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for master")
		}
		k.Master.IP = ip
		k.UsedIPs[k.Master.Node] = append(k.UsedIPs[k.Master.Node], ip)
	}
	for idx, w := range k.Workers {
		workerNodeRange := k.NodesIPRange[w.Node]
		if w.IP != "" && workerNodeRange.Contains(net.ParseIP(w.IP)) {
			continue
		}
		ip, err := getK8sFreeIP(workerNodeRange, k.UsedIPs[w.Node])
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for worker")
		}
		k.Workers[idx].IP = ip
		k.UsedIPs[w.Node] = append(k.UsedIPs[w.Node], ip)
	}
	return nil
}
func (k *K8sDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	err := k.assignNodesIPs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	deployments := make(map[uint32]gridtypes.Deployment)
	nodeWorkloads := make(map[uint32][]gridtypes.Workload)
	masterWorkloads := k.Master.GenerateK8sWorkload(k, "")
	nodeWorkloads[k.Master.Node] = append(nodeWorkloads[k.Master.Node], masterWorkloads...)
	for _, w := range k.Workers {
		workerWorkloads := w.GenerateK8sWorkload(k, k.Master.IP)
		nodeWorkloads[w.Node] = append(nodeWorkloads[w.Node], workerWorkloads...)
	}

	for node, ws := range nodeWorkloads {
		dl := gridtypes.Deployment{
			Version: 0,
			TwinID:  uint32(k.APIClient.twin_id), //LocalTwin,
			// this contract id must match the one on substrate
			Workloads: ws,
			SignatureRequirement: gridtypes.SignatureRequirement{
				WeightRequired: 1,
				Requests: []gridtypes.SignatureRequest{
					{
						TwinID: k.APIClient.twin_id,
						Weight: 1,
					},
				},
			},
		}
		deployments[node] = dl
	}
	return deployments, nil
}

func (k *K8sDeployer) GetOldDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	return getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
}

func (k *K8sDeployer) Valid(ctx context.Context) error {

	names := make(map[string]bool)
	names[k.Master.Name] = true
	for _, w := range k.Workers {
		if _, ok := names[w.Name]; ok {
			return fmt.Errorf("k8s workers and master must have unique names: %s occured more than once", w.Name)
		}
		names[w.Name] = true
	}
	machines := k.Workers
	if k.Master != nil {
		machines = append(machines, *k.Master)
	}

	for _, w := range machines {
		cl, err := k.ncPool.getNodeClient(w.Node)
		if err != nil {
			return errors.Wrap(err, "couldn't get node client")
		}
		if err := isNodeUp(ctx, cl); err != nil {
			return fmt.Errorf("couldn't reach node %d: %w", w.Node, err)
		}
	}
	return nil
}

func (k *K8sDeployer) Deploy(ctx context.Context) error {
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	oldDeployments, err := k.GetOldDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get old deployments data")
	}
	currentDeployments, err := deployDeployments(ctx, oldDeployments, newDeployments, k.ncPool, k.APIClient, true)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func (k *K8sDeployer) Cancel(ctx context.Context) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	oldDeployments, err := k.GetOldDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get old deployments data")
	}
	currentDeployments, err := deployDeployments(ctx, oldDeployments, newDeployments, k.ncPool, k.APIClient, false)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func printDeployments(dls map[uint32]gridtypes.Deployment) {
	for node, dl := range dls {
		log.Printf("node id: %d\n", node)
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(dl)
	}
}

func (k *K8sDeployer) updateState(ctx context.Context, currentDeploymentIDs map[uint32]uint64) error {
	log.Printf("current deployments\n")
	k.NodeDeploymentID = currentDeploymentIDs
	currentDeployments, err := getDeploymentObjects(ctx, currentDeploymentIDs, k.ncPool)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}
	printDeployments(currentDeployments)
	publicIPs := make(map[string]string)
	privateIPs := make(map[string]string)
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.PublicIPType {
				d := PubIPData{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("error unmarshalling json: %s\n", err)
					continue
				}
				publicIPs[string(w.Name)] = d.IP
			} else if w.Type == zos.ZMachineType {
				d, err := w.WorkloadData()
				if err != nil {
					log.Printf("error loading machine data: %s\n", err)
					continue
				}
				privateIPs[string(w.Name)] = d.(*zos.ZMachine).Network.Interfaces[0].IP.String()
			}
		}
	}
	masterIPName := fmt.Sprintf("%sip", k.Master.Name)
	if ip, ok := publicIPs[masterIPName]; ok {
		k.Master.ComputedIP = ip
	} else {
		k.Master.ComputedIP = ""
	}
	private, ok := privateIPs[string(k.Master.Name)]
	if ok {
		k.Master.IP = private
	} else {
		k.Master.IP = ""
	}

	for idx, w := range k.Workers {
		workerIPName := fmt.Sprintf("%sip", w.Name)
		if ip, ok := publicIPs[workerIPName]; ok {
			k.Workers[idx].ComputedIP = ip
		} else {
			k.Workers[idx].ComputedIP = ""
		}
		private, ok := privateIPs[string(w.Name)]
		if ok {
			k.Workers[idx].IP = private
		} else {
			k.Workers[idx].IP = ""
		}
	}
	log.Printf("Current state after updatestate %v\n", k)
	return nil
}

func (k *K8sDeployer) removeDeletedContracts(ctx context.Context) error {
	nodeDeploymentID := make(map[uint32]uint64)
	for nodeID, deploymentID := range k.NodeDeploymentID {
		cont, err := k.APIClient.sub.GetContract(deploymentID)
		if err != nil {
			return errors.Wrap(err, "failed to get deployments")
		}
		if !cont.State.IsDeleted {
			nodeDeploymentID[nodeID] = deploymentID
		}
	}
	k.NodeDeploymentID = nodeDeploymentID
	return nil
}

func (k *K8sDeployer) updateFromRemote(ctx context.Context) error {
	if err := k.removeDeletedContracts(ctx); err != nil {
		return errors.Wrap(err, "failed to remove deleted contracts")
	}
	currentDeployments, err := getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
	if err != nil {
		return errors.Wrap(err, "failed to fetch remote deployments")
	}
	log.Printf("calling updateFromRemote")
	printDeployments(currentDeployments)
	keyUpdated, tokenUpdated, networkUpdated := false, false, false
	// calculate k's properties from the currently deployed deployments
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				d, err := w.WorkloadData()
				if err != nil {
					log.Printf("failed to get workload data %s", err)
				}
				SSHKey := d.(*zos.ZMachine).Env["SSH_KEY"]
				token := d.(*zos.ZMachine).Env["K3S_TOKEN"]
				networkName := string(d.(*zos.ZMachine).Network.Interfaces[0].Network)
				if !keyUpdated && SSHKey != k.SSHKey {
					k.SSHKey = SSHKey
					keyUpdated = true
				}
				if !tokenUpdated && token != k.Token {
					k.Token = token
					tokenUpdated = true
				}
				if !networkUpdated && networkName != k.NetworkName {
					k.NetworkName = networkName
					networkUpdated = true
				}
			}
		}
	}

	log.Printf("calling updateFromRemote1")
	nodeDeploymentID := make(map[uint32]uint64)
	for node, dl := range currentDeployments {
		nodeDeploymentID[node] = dl.ContractID
	}
	k.NodeDeploymentID = nodeDeploymentID
	// maps from workload name to (public ip, node id, disk size, actual workload)
	workloadNodeID := make(map[string]uint32)
	workloadDiskSize := make(map[string]int)
	workloadComputedIP := make(map[string]string)
	workloadObj := make(map[string]gridtypes.Workload)

	publicIPs := make(map[string]string)
	diskSize := make(map[string]int)
	for node, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				workloadNodeID[string(w.Name)] = node
				workloadObj[string(w.Name)] = w

			} else if w.Type == zos.PublicIPType {
				d := PubIPData{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("failed to load pubip data %s", err)
					continue
				}
				publicIPs[string(w.Name)] = d.IP
			} else if w.Type == zos.ZMountType {
				d, err := w.WorkloadData()
				if err != nil {
					log.Printf("failed to load disk data %s", err)
					continue
				}
				diskSize[string(w.Name)] = int(d.(*zos.ZMount).Size / gridtypes.Gigabyte)
			}
		}
	}
	log.Printf("calling updateFromRemote2")
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				publicIPKey := fmt.Sprintf("%sip", w.Name)
				diskKey := fmt.Sprintf("%sdisk", w.Name)
				workloadDiskSize[string(w.Name)] = diskSize[diskKey]
				workloadComputedIP[string(w.Name)] = publicIPs[publicIPKey]
			}
		}
	}
	log.Printf("calling updateFromRemote3")
	// update master
	masterNodeID, ok := workloadNodeID[k.Master.Name]
	if !ok {
		k.Master = nil
	} else {
		masterWorkload := workloadObj[k.Master.Name]
		masterIP, ipExists := workloadComputedIP[k.Master.Name]
		if !ipExists {
			masterIP = ""
		}
		masterDiskSize := workloadDiskSize[k.Master.Name]

		m, err := NewK8sNodeDataFromWorkload(masterWorkload, masterNodeID, masterDiskSize, masterIP)
		if err != nil {
			return errors.Wrap(err, "failed to get master data from workload")
		}
		k.Master = &m
	}
	log.Printf("calling updateFromRemote4")
	// update workers
	workers := make([]K8sNodeData, 0)
	for _, w := range k.Workers {
		workerNodeID, ok := workloadNodeID[w.Name]
		if !ok {
			// worker doesn't exist in any deployment, skip it
			continue
		}
		delete(workloadNodeID, w.Name)
		workerWorkload := workloadObj[w.Name]
		workerIP, ipExists := workloadComputedIP[w.Name]
		if !ipExists {
			workerIP = ""
		}
		workerDiskSize := workloadDiskSize[w.Name]
		w, err := NewK8sNodeDataFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
	log.Printf("calling updateFromRemote5")
	// add missing workers (in case of failed deletions)
	for name, workerNodeID := range workloadNodeID {
		if name == k.Master.Name {
			continue
		}
		workerWorkload := workloadObj[name]
		workerIP, ipExists := workloadComputedIP[name]
		if !ipExists {
			workerIP = ""
		}
		workerDiskSize := workloadDiskSize[name]
		w, err := NewK8sNodeDataFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
	log.Printf("calling updateFromRemote6")
	k.Workers = workers
	log.Printf("after updateFromRemote\n")
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	enc.Encode(k)

	return nil
}

func (k *K8sNodeData) GenerateK8sWorkload(deployer *K8sDeployer, masterIP string) []gridtypes.Workload {
	diskName := fmt.Sprintf("%sdisk", k.Name)
	workloads := make([]gridtypes.Workload, 0)
	diskWorkload := gridtypes.Workload{
		Name:        gridtypes.Name(diskName),
		Version:     0,
		Type:        zos.ZMountType,
		Description: "",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(k.DiskSize) * gridtypes.Gigabyte,
		}),
	}
	workloads = append(workloads, diskWorkload)
	publicIPName := ""
	if k.PublicIP {
		publicIPName = fmt.Sprintf("%sip", k.Name)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName))
	}
	envVars := map[string]string{
		"SSH_KEY":           deployer.SSHKey,
		"K3S_TOKEN":         deployer.Token,
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     k.Name,
		"K3S_URL":           "",
	}
	if masterIP != "" {
		envVars["K3S_URL"] = fmt.Sprintf("https://%s:6443", masterIP)
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(k.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: k.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(deployer.NetworkName),
						IP:      net.ParseIP(k.IP),
					},
				},
				PublicIP: gridtypes.Name(publicIPName),
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(k.Cpu),
				Memory: gridtypes.Unit(uint(k.Memory)) * gridtypes.Megabyte,
			},
			Entrypoint: "/sbin/zinit init",
			Mounts: []zos.MachineMount{
				{Name: gridtypes.Name(diskName), Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}
	workloads = append(workloads, workload)

	return workloads
}

func getK8sFreeIP(ipRange gridtypes.IPNet, usedIPs []string) (string, error) {
	i := 254
	l := len(ipRange.IP)
	for i >= 2 {
		ip := ipNet(ipRange.IP[l-4], ipRange.IP[l-3], ipRange.IP[l-2], byte(i), 32)
		ipStr := fmt.Sprintf("%d.%d.%d.%d", ip.IP[l-4], ip.IP[l-3], ip.IP[l-2], ip.IP[l-1])
		log.Printf("ip string: %s\n", ipStr)
		if !isInStr(usedIPs, ipStr) {
			return ipStr, nil
		}
		i -= 1
	}
	return "", errors.New("all ips are used")
}

func resourceK8sCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Valid(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}

	err = deployer.Deploy(ctx)
	if err != nil {
		if len(deployer.NodeDeploymentID) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	deployer.storeState(d)
	d.SetId(uuid.New().String())
	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Valid(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}

	err = deployer.Deploy(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func resourceK8sRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Valid(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}

	err = deployer.updateFromRemote(ctx)
	log.Printf("read updateFromRemote err: %s\n", err)
	if err != nil {
		return diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func resourceK8sDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Valid(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}

	err = deployer.Cancel(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
	} else {
		deployer.storeState(d)
	}
	return diags
}
