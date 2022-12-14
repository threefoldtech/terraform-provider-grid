package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func resourceKubernetes() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Kubernetes resource.",

		CreateContext: resourceK8sCreate,
		ReadContext:   resourceK8sRead,
		UpdateContext: resourceK8sUpdate,
		DeleteContext: resourceK8sDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Instance name",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Kubernetes",
			},
			"capacity_deployment_map": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each capacityID to its deployment id",
			},
			"network_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The network name to deploy the cluster on",
			},
			"ssh_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSH key to access the cluster nodes",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The cluster secret token",
			},
			"nodes_ip_range": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Network IP ranges of nodes in the cluster (usually assigned from grid_network.<network-resource-name>.nodes_ip_range)",
			},
			"master": {
				MaxItems: 1,
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Master name",
						},
						"node": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Node ID",
						},
						"capacity_id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Capacity reservation contract id from capacity reserver",
						},
						"disk_size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Data disk size in GBs",
						},
						"publicip": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "true to enable public ip reservation",
						},
						"publicip6": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "true to enable public ipv6 reservation",
						},
						"flist": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
						},
						"flist_checksum": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "if present, the flist is rejected if it has a different hash. the flist hash can be found by append",
						},
						"computedip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public IP",
						},
						"computedip6": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public IPv6",
						},
						"ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The private IP (computed from nodes_ip_range)",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Number of VCPUs",
						},
						"memory": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Memory size",
						},
						"planetary": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Enable Yggdrasil allocation",
						},
						"ygg_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Allocated Yggdrasil IP",
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
							Default:  "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
						},
						"flist_checksum": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "if present, the flist is rejected if it has a different hash. the flist hash can be found by append",
						},
						"disk_size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Data disk size in GBs",
						},
						"node": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Node ID",
						},
						"capacity_id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Capacity reservation contract id from capacity reserver",
						},
						"publicip": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "true to enable public ip reservation",
						},
						"computedip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public ip",
						},
						"publicip6": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "true to enable public ipv6 reservation",
						},
						"computedip6": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public ipv6",
						},
						"ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The private IP (computed from nodes_ip_range)",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Number of VCPUs",
						},
						"memory": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Memory size",
						},
						"planetary": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Enable Yggdrasil allocation",
						},
						"ygg_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Allocated Yggdrasil IP",
						},
					},
				},
			},
		},
	}
}

type K8sNodeData struct {
	Name          string
	Node          uint32
	CapacityId    uint64
	DiskSize      int
	PublicIP      bool
	PublicIP6     bool
	Planetary     bool
	Flist         string
	FlistChecksum string
	ComputedIP    string
	ComputedIP6   string
	YggIP         string
	IP            string
	Cpu           int
	Memory        int
}

type K8sDeployer struct {
	Master                *K8sNodeData
	Workers               []K8sNodeData
	NodesIPRange          map[uint32]gridtypes.IPNet
	Token                 string
	SSHKey                string
	NetworkName           string
	CapacityDeploymentMap map[uint64]uint64

	APIClient *apiClient

	NodeUsedIPs map[uint32][]byte
	ncPool      *client.NodeClientPool
	d           *schema.ResourceData
	deployer    deployer.Deployer
}

func NewK8sNodeData(m map[string]interface{}, sub *substrate.Substrate) (K8sNodeData, error) {
	capacityId := uint64(m["capacity_id"].(int))
	nodeId, err := getNodeIdByCapacityId(sub, capacityId)
	if err != nil {
		return K8sNodeData{}, errors.Wrapf(err, "can't get nodeId for capacityId: %d", capacityId)
	}
	return K8sNodeData{
		Name:          m["name"].(string),
		Node:          uint32(nodeId),
		CapacityId:    capacityId,
		DiskSize:      m["disk_size"].(int),
		PublicIP:      m["publicip"].(bool),
		PublicIP6:     m["publicip6"].(bool),
		Planetary:     m["planetary"].(bool),
		Flist:         m["flist"].(string),
		FlistChecksum: m["flist_checksum"].(string),
		ComputedIP:    m["computedip"].(string),
		ComputedIP6:   m["computedip6"].(string),
		YggIP:         m["ygg_ip"].(string),
		IP:            m["ip"].(string),
		Cpu:           m["cpu"].(int),
		Memory:        m["memory"].(int),
	}, nil
}

func NewK8sNodeDataFromWorkload(w gridtypes.Workload, nodeID uint32, capacityId uint64, diskSize int, computedIP string, computedIP6 string) (K8sNodeData, error) {
	var k K8sNodeData
	data, err := w.WorkloadData()
	if err != nil {
		return k, err
	}
	d := data.(*zos.ZMachine)
	var result zos.ZMachineResult
	err = w.Result.Unmarshal(&result)
	if err != nil {
		return k, err
	}
	k = K8sNodeData{
		Name:        string(w.Name),
		Node:        nodeID,
		CapacityId:  capacityId,
		DiskSize:    diskSize,
		PublicIP:    computedIP != "",
		PublicIP6:   computedIP6 != "",
		Planetary:   result.YggIP != "",
		Flist:       d.FList,
		ComputedIP:  computedIP,
		ComputedIP6: computedIP6,
		YggIP:       result.YggIP,
		IP:          d.Network.Interfaces[0].IP.String(),
		Cpu:         int(d.ComputeCapacity.CPU),
		Memory:      int(d.ComputeCapacity.Memory / gridtypes.Megabyte),
	}
	return k, nil
}

func NewK8sDeployer(d *schema.ResourceData, apiClient *apiClient) (K8sDeployer, error) {
	networkName := d.Get("network_name").(string)
	ns := apiClient.state.GetNetworkState()
	network := ns.GetNetwork(networkName)
	master, err := NewK8sNodeData(d.Get("master").([]interface{})[0].(map[string]interface{}), apiClient.substrateConn)

	if err != nil {
		return K8sDeployer{}, errors.Wrap(err, "couldn't construct master node data")
	}
	masterNodeId, err := getNodeIdByCapacityId(apiClient.substrateConn, master.CapacityId)
	if err != nil {
		return K8sDeployer{}, errors.Wrapf(err, "couldn't get master node Id, CapacityId: ", master.CapacityId)
	}
	master.Node = masterNodeId
	workers := make([]K8sNodeData, 0)
	usedIPs := make(map[uint32][]byte)

	if master.IP != "" {
		usedIPs[master.Node] = append(usedIPs[master.Node], net.ParseIP(master.IP)[3])
	}
	usedIPs[master.Node] = append(usedIPs[master.Node], network.GetNodeIPsList(master.Node)...)
	for _, w := range d.Get("workers").([]interface{}) {
		data, err := NewK8sNodeData(w.(map[string]interface{}), apiClient.substrateConn)
		if err != nil {
			return K8sDeployer{}, errors.Wrapf(err, "couldn't construct worker node data capacityId: %d", master.CapacityId)
		}
		workerNodeId, err := getNodeIdByCapacityId(apiClient.substrateConn, data.CapacityId)
		if err != nil {
			return K8sDeployer{}, errors.Wrapf(err, "couldn't get worker nodeId, capacityId: %d", data.CapacityId)
		}
		data.Node = workerNodeId
		workers = append(workers, data)
		if data.IP != "" {
			usedIPs[data.Node] = append(usedIPs[data.Node], net.ParseIP(data.IP)[3])
			usedIPs[data.Node] = append(usedIPs[data.Node], network.GetNodeIPsList(data.Node)...)
		}

	}
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	masterIPRange, err := gridtypes.ParseIPNet(network.GetNodeSubnet(master.Node))
	if err != nil {
		return K8sDeployer{}, errors.Wrap(err, "couldn't parse master node ip range")
	}
	nodesIPRange[master.Node] = masterIPRange
	for _, worker := range workers {
		nodesIPRange[worker.Node], err = gridtypes.ParseIPNet(network.GetNodeSubnet(worker.Node))
		if err != nil {
			return K8sDeployer{}, errors.Wrapf(err, "couldn't parse worker node (%d) ip range", worker.Node)
		}
	}
	capacityDeploymentRaw := d.Get("capacity_deployment_map").(map[string]interface{})
	capacityDeploymentMap := make(map[uint64]uint64)
	for capId, dlId := range capacityDeploymentRaw {
		capacityId, err := strconv.ParseUint(capId, 10, 64)
		if err != nil {
			return K8sDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		deploymentID := uint64(dlId.(int))
		capacityDeploymentMap[capacityId] = deploymentID
	}

	pool := client.NewNodeClientPool(apiClient.rmb)
	deploymentData := DeploymentData{
		Name:        d.Get("name").(string),
		Type:        "kubernetes",
		ProjectName: d.Get("solution_type").(string),
	}
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}
	deployer := K8sDeployer{
		Master:                &master,
		Workers:               workers,
		Token:                 d.Get("token").(string),
		SSHKey:                d.Get("ssh_key").(string),
		NetworkName:           d.Get("network_name").(string),
		CapacityDeploymentMap: capacityDeploymentMap,
		NodeUsedIPs:           usedIPs,
		NodesIPRange:          nodesIPRange,
		APIClient:             apiClient,
		ncPool:                pool,
		d:                     d,
		deployer:              deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, pool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

func (k *K8sNodeData) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = k.Name
	res["node"] = uint64(k.Node)
	res["capacity_id"] = uint64(k.CapacityId)
	res["disk_size"] = k.DiskSize
	res["publicip"] = k.PublicIP
	res["publicip6"] = k.PublicIP6
	res["planetary"] = k.Planetary
	res["flist"] = k.Flist
	res["computedip"] = k.ComputedIP
	res["computedip6"] = k.ComputedIP6
	res["ygg_ip"] = k.YggIP
	res["ip"] = k.IP
	res["cpu"] = k.Cpu
	res["memory"] = k.Memory
	return res
}

// invalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *K8sDeployer) invalidateBrokenAttributes(sub *substrate.Substrate) error {
	newWorkers := make([]K8sNodeData, 0)

	validNodes := make(map[uint32]struct{})
	for capacityId := range k.CapacityDeploymentMap {
		nodeId, err := getNodeIdByCapacityId(sub, capacityId)
		if err != nil {
			return errors.Wrapf(err, "couldn't get node id for capacity Id", capacityId)
		}
		contract, err := sub.GetContract(capacityId)
		if (err == nil && !contract.State.IsCreated) || errors.Is(err, substrate.ErrNotFound) {
			delete(k.CapacityDeploymentMap, capacityId)
			delete(k.NodesIPRange, nodeId)
		} else if err != nil {
			return errors.Wrapf(err, "couldn't get capacity contract with ID: %d", capacityId)
		} else {
			validNodes[nodeId] = struct{}{}
		}

	}
	if _, ok := validNodes[k.Master.Node]; !ok {
		k.Master = &K8sNodeData{}
	}
	for _, worker := range k.Workers {
		if _, ok := validNodes[worker.Node]; ok {
			newWorkers = append(newWorkers, worker)
		}
	}
	k.Workers = newWorkers
	return nil
}

func (d *K8sDeployer) retainChecksums(workers []interface{}, master interface{}) {
	checksumMap := make(map[string]string)
	checksumMap[d.Master.Name] = d.Master.FlistChecksum
	for _, w := range d.Workers {
		checksumMap[w.Name] = w.FlistChecksum
	}
	typed := master.(map[string]interface{})
	typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	for _, w := range workers {
		typed := w.(map[string]interface{})
		typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	}
}

func (k *K8sDeployer) storeState(d *schema.ResourceData, cl *apiClient) error {
	workers := make([]interface{}, 0)
	for _, w := range k.Workers {
		workers = append(workers, w.Dictify())
	}
	capacityDeploymentMap := make(map[string]interface{})
	for capacityId, deploymentId := range k.CapacityDeploymentMap {
		capacityDeploymentMap[fmt.Sprint(capacityId)] = int(deploymentId)
	}
	log.Printf("master data: %v\n", k.Master)
	if k.Master == nil {
		k.Master = &K8sNodeData{}
	}
	master := k.Master.Dictify()
	k.retainChecksums(workers, master)

	l := []interface{}{master}
	k.updateNetworkState(cl.substrateConn, d, cl.state)
	err := errSet{}
	err.Push(d.Set("master", l))
	err.Push(d.Set("workers", workers))
	err.Push(d.Set("token", k.Token))
	err.Push(d.Set("ssh_key", k.SSHKey))
	err.Push(d.Set("network_name", k.NetworkName))
	err.Push(d.Set("capacity_deployment_map", capacityDeploymentMap))
	return err.error()
}

func (k *K8sDeployer) updateNetworkState(sub *substrate.Substrate, d *schema.ResourceData, state state.StateI) {
	ns := state.GetNetworkState()
	network := ns.GetNetwork(k.NetworkName)
	before, _ := d.GetChange("capacity_deployment_map")
	for capacityIdStr, deploymentID := range before.(map[string]interface{}) {
		capacityId, err := strconv.ParseUint(capacityIdStr, 10, 64)
		if err != nil {
			log.Printf("error converting node id string to int: %+v", err)
			continue
		}
		deploymentIDStr := fmt.Sprint(deploymentID.(int))
		nodeId, err := getNodeIdByCapacityId(sub, capacityId)
		// shouldn't panic if we failed to get the node to free the ip
		if err != nil {
			continue
		}
		network.DeleteDeployment(nodeId, deploymentIDStr)
	}
	// remove old ips
	masterDl := k.CapacityDeploymentMap[k.Master.CapacityId]
	network.DeleteDeployment(k.Master.Node, fmt.Sprint(masterDl))
	for _, worker := range k.Workers {
		workerDl := k.CapacityDeploymentMap[worker.CapacityId]
		network.DeleteDeployment(worker.Node, fmt.Sprint(workerDl))
	}

	// append new ips
	masterNodeIPs := network.GetDeploymentIPs(k.Master.Node, fmt.Sprint(masterDl))
	masterIP := net.ParseIP(k.Master.IP)
	if masterIP == nil {
		log.Printf("couldn't parse master ip, value: %s", k.Master.IP)
	} else {
		masterNodeIPs = append(masterNodeIPs, masterIP.To4()[3])
	}
	network.SetDeploymentIPs(k.Master.Node, fmt.Sprint(masterDl), masterNodeIPs)
	for _, worker := range k.Workers {
		workerDl := k.CapacityDeploymentMap[worker.CapacityId]
		workerNodeIPs := network.GetDeploymentIPs(worker.Node, fmt.Sprint(workerDl))
		workerIP := net.ParseIP(worker.IP)
		if workerIP == nil {
			log.Printf("couldn't parse worker ip at node (%d)", worker.Node)
		} else {
			workerNodeIPs = append(workerNodeIPs, workerIP.To4()[3])
		}
		network.SetDeploymentIPs(worker.Node, fmt.Sprint(workerDl), workerNodeIPs)
	}
}

func (k *K8sDeployer) assignNodesIPs() error {
	// TODO: when a k8s node changes its zos node, remove its ip from the used ones. better at the beginning
	masterNodeRange := k.NodesIPRange[k.Master.Node]
	if k.Master.IP == "" || !masterNodeRange.Contains(net.ParseIP(k.Master.IP)) {
		ip, err := k.getK8sFreeIP(masterNodeRange, k.Master.Node)
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for master")
		}
		k.Master.IP = ip
	}
	for idx, w := range k.Workers {
		workerNodeRange := k.NodesIPRange[w.Node]
		if w.IP != "" && workerNodeRange.Contains(net.ParseIP(w.IP)) {
			continue
		}
		ip, err := k.getK8sFreeIP(workerNodeRange, w.Node)
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for worker")
		}
		k.Workers[idx].IP = ip
	}
	return nil
}
func (k *K8sDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint64]gridtypes.Deployment, error) {
	err := k.assignNodesIPs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	deployments := make(map[uint64]gridtypes.Deployment)
	capacityWorkloadsMap := make(map[uint64][]gridtypes.Workload)
	masterWorkloads := k.Master.GenerateK8sWorkload(k, "")
	capacityWorkloadsMap[k.Master.CapacityId] = append(capacityWorkloadsMap[k.Master.CapacityId], masterWorkloads...)
	for _, w := range k.Workers {
		workerWorkloads := w.GenerateK8sWorkload(k, k.Master.IP)
		capacityWorkloadsMap[w.CapacityId] = append(capacityWorkloadsMap[w.CapacityId], workerWorkloads...)
	}

	for capacityId, ws := range capacityWorkloadsMap {
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
		deployments[capacityId] = dl
	}
	return deployments, nil
}

func (d *K8sDeployer) validateChecksums() error {
	nodes := append(d.Workers, *d.Master)
	for _, vm := range nodes {
		if vm.FlistChecksum == "" {
			continue
		}
		checksum, err := getFlistChecksum(vm.Flist)
		if err != nil {
			return errors.Wrapf(err, "couldn't get flist %s hash", vm.Flist)
		}
		if vm.FlistChecksum != checksum {
			return fmt.Errorf("passed checksum %s of %s doesn't match %s returned from %s",
				vm.FlistChecksum,
				vm.Name,
				checksum,
				flistChecksumURL(vm.Flist),
			)
		}
	}
	return nil
}

func (k *K8sDeployer) ValidateNames(ctx context.Context) error {

	names := make(map[string]bool)
	names[k.Master.Name] = true
	for _, w := range k.Workers {
		if _, ok := names[w.Name]; ok {
			return fmt.Errorf("k8s workers and master must have unique names: %s occured more than once", w.Name)
		}
		names[w.Name] = true
	}
	return nil
}

func (k *K8sDeployer) ValidateIPranges(ctx context.Context) error {

	if _, ok := k.NodesIPRange[k.Master.Node]; !ok {
		return fmt.Errorf("the master node %d doesn't exist in the network's ip ranges", k.Master.Node)
	}
	for _, w := range k.Workers {
		if _, ok := k.NodesIPRange[w.Node]; !ok {
			return fmt.Errorf("the node with id %d in worker %s doesn't exist in the network's ip ranges", w.Node, w.Name)
		}
	}
	return nil
}

func (k *K8sDeployer) validateToken(ctx context.Context) error {
	if k.Token == "" {
		return errors.New("empty token is now allowed")
	}

	is_alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(k.Token)
	if !is_alphanumeric {
		return errors.New("token should be alphanumeric")
	}

	return nil
}

func (k *K8sDeployer) Validate(ctx context.Context, sub *substrate.Substrate) error {
	if err := k.validateToken(ctx); err != nil {
		return err
	}
	if err := validateAccountMoneyForExtrinsics(sub, k.APIClient.identity); err != nil {
		return err
	}
	if err := k.ValidateNames(ctx); err != nil {
		return err
	}
	if err := k.ValidateIPranges(ctx); err != nil {
		return err
	}
	nodes := make([]uint32, 0)
	nodes = append(nodes, k.Master.Node)
	for _, w := range k.Workers {
		nodes = append(nodes, w.Node)

	}
	return isNodesUp(ctx, sub, nodes, k.ncPool)
}

func (k *K8sDeployer) Deploy(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, cl *apiClient) error {
	if err := k.validateChecksums(); err != nil {
		return err
	}
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
	if err := k.updateState(ctx, sub, currentDeployments, d, cl); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func (k *K8sDeployer) Cancel(ctx context.Context, sub *substrate.Substrate, d *schema.ResourceData, cl *apiClient) error {
	newDeployments := make(map[uint64]gridtypes.Deployment)

	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
	if err != nil {
		return errors.Wrapf(err, "couldn't cancel k8s deployment")
	}
	// remove used ips
	k.removeUsedIPsFromLocalState(cl)

	if err := k.updateState(ctx, sub, currentDeployments, d, cl); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func printDeployments(dls map[uint64]gridtypes.Deployment) {
	for capacityId, dl := range dls {
		log.Printf("capacityId: %d\n", capacityId)
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(dl)
	}
}

func (k *K8sDeployer) removeUsedIPsFromLocalState(cl *apiClient) {
	ns := cl.state.GetNetworkState()
	network := ns.GetNetwork(k.NetworkName)
	masterDl := k.CapacityDeploymentMap[k.Master.CapacityId]
	network.DeleteDeployment(k.Master.Node, fmt.Sprint(masterDl))
	for _, worker := range k.Workers {
		workerDl := k.CapacityDeploymentMap[worker.CapacityId]
		network.DeleteDeployment(worker.Node, fmt.Sprint(workerDl))
	}
}

func (k *K8sDeployer) updateState(ctx context.Context, sub *substrate.Substrate, currentDeploymentIDs map[uint64]uint64, d *schema.ResourceData, cl *apiClient) error {
	log.Printf("current deployments\n")
	k.CapacityDeploymentMap = currentDeploymentIDs
	currentDeployments, err := k.deployer.GetDeploymentObjects(ctx, sub, currentDeploymentIDs)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}
	printDeployments(currentDeployments)
	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	yggIPs := make(map[string]string)
	privateIPs := make(map[string]string)
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.PublicIPType {
				d := zos.PublicIPResult{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("error unmarshalling json: %s\n", err)
					continue
				}
				publicIPs[string(w.Name)] = d.IP.String()
				publicIP6s[string(w.Name)] = d.IPv6.String()
			} else if w.Type == zos.ZMachineType {
				d, err := w.WorkloadData()
				if err != nil {
					log.Printf("error loading machine data: %s\n", err)
					continue
				}
				privateIPs[string(w.Name)] = d.(*zos.ZMachine).Network.Interfaces[0].IP.String()

				var result zos.ZMachineResult
				if err := w.Result.Unmarshal(&result); err != nil {
					log.Printf("error loading machine result: %s\n", err)
				}
				yggIPs[string(w.Name)] = result.YggIP
			}
		}
	}
	masterIPName := fmt.Sprintf("%sip", k.Master.Name)
	k.Master.ComputedIP = publicIPs[masterIPName]
	k.Master.ComputedIP6 = publicIP6s[masterIPName]
	k.Master.IP = privateIPs[string(k.Master.Name)]
	k.Master.YggIP = yggIPs[string(k.Master.Name)]

	for idx, w := range k.Workers {
		workerIPName := fmt.Sprintf("%sip", w.Name)
		k.Workers[idx].ComputedIP = publicIPs[workerIPName]
		k.Workers[idx].ComputedIP = publicIP6s[workerIPName]
		k.Workers[idx].IP = privateIPs[string(w.Name)]
		k.Workers[idx].YggIP = yggIPs[string(w.Name)]
	}
	k.updateNetworkState(sub, d, cl.state)
	log.Printf("Current state after updatestate %v\n", k)
	return nil
}

func (k *K8sDeployer) removeDeletedContracts(ctx context.Context, sub *substrate.Substrate) error {
	capacityDeploymentMap := make(map[uint64]uint64)
	for capacityId, deploymentID := range k.CapacityDeploymentMap {

		cont, err := sub.GetContract(capacityId)
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity contract %d", capacityId)
		}

		if !cont.State.IsDeleted {
			_, err = sub.GetDeployment(deploymentID)
			if err != nil {
				if errors.Is(err, substrate.ErrNotFound) {
					continue
				}
				return errors.Wrapf(err, "failed to get deployment with id: %d", deploymentID)
			}
			capacityDeploymentMap[capacityId] = deploymentID
		}

	}
	k.CapacityDeploymentMap = capacityDeploymentMap
	return nil
}

func (k *K8sDeployer) updateFromRemote(ctx context.Context, sub *substrate.Substrate) error {
	if err := k.removeDeletedContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "failed to remove deleted contracts")
	}
	currentDeployments, err := k.deployer.GetDeploymentObjects(ctx, sub, k.CapacityDeploymentMap)
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
					return errors.Wrapf(err, "failed to get workload data deploymentId: %d", dl.DeploymentID)
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

	capacityDeploymentMap := make(map[uint64]uint64)
	for capacityId, dl := range currentDeployments {
		capacityDeploymentMap[capacityId] = dl.DeploymentID.U64()
	}
	k.CapacityDeploymentMap = capacityDeploymentMap
	// maps from workload name to (public ip, node id, disk size, actual workload)
	workloadCapacityMap := make(map[string]uint64)
	workloadDiskSize := make(map[string]int)
	workloadComputedIP := make(map[string]string)
	workloadComputedIP6 := make(map[string]string)
	workloadObj := make(map[string]gridtypes.Workload)

	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	diskSize := make(map[string]int)
	for capacityId, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				workloadCapacityMap[string(w.Name)] = capacityId
				workloadObj[string(w.Name)] = w

			} else if w.Type == zos.PublicIPType {
				d := zos.PublicIPResult{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					return errors.Wrapf(err, "failed to load pubip data: %d", dl.DeploymentID)
				}
				publicIPs[string(w.Name)] = d.IP.String()
				publicIP6s[string(w.Name)] = d.IPv6.String()
			} else if w.Type == zos.ZMountType {
				d, err := w.WorkloadData()
				if err != nil {
					return errors.Wrapf(err, "failed to load disk data: %d", dl.DeploymentID)
				}
				diskSize[string(w.Name)] = int(d.(*zos.ZMount).Size / gridtypes.Gigabyte)
			}
		}
	}
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				publicIPKey := fmt.Sprintf("%sip", w.Name)
				diskKey := fmt.Sprintf("%sdisk", w.Name)
				workloadDiskSize[string(w.Name)] = diskSize[diskKey]
				workloadComputedIP[string(w.Name)] = publicIPs[publicIPKey]
				workloadComputedIP6[string(w.Name)] = publicIP6s[publicIPKey]
			}
		}
	}
	// update master
	masterCapacityId, ok := workloadCapacityMap[k.Master.Name]
	if !ok {
		k.Master = nil
	} else {
		masterWorkload := workloadObj[k.Master.Name]
		masterIP := workloadComputedIP[k.Master.Name]
		masterIP6 := workloadComputedIP6[k.Master.Name]
		masterDiskSize := workloadDiskSize[k.Master.Name]
		masterNodeId, err := getNodeIdByCapacityId(sub, masterCapacityId)
		if err != nil {
			errors.Wrapf(err, "couldn't get master nodeID, capacityId: %d", masterCapacityId)
		}
		m, err := NewK8sNodeDataFromWorkload(masterWorkload, masterNodeId, masterCapacityId, masterDiskSize, masterIP, masterIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get master data from workload")
		}
		k.Master = &m
	}
	// update workers
	workers := make([]K8sNodeData, 0)
	for _, w := range k.Workers {
		workerCapacityId, ok := workloadCapacityMap[w.Name]
		if !ok {
			// worker doesn't exist in any deployment, skip it
			continue
		}
		delete(workloadCapacityMap, w.Name)
		workerWorkload := workloadObj[w.Name]
		workerIP := workloadComputedIP[w.Name]
		workerIP6 := workloadComputedIP6[w.Name]
		workerDiskSize := workloadDiskSize[w.Name]
		workerNodeId, err := getNodeIdByCapacityId(sub, workerCapacityId)
		if err != nil {
			errors.Wrapf(err, "couldn't get worker nodeID, capacityId: %d", workerCapacityId)
		}
		w, err := NewK8sNodeDataFromWorkload(workerWorkload, workerNodeId, workerCapacityId, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
	// add missing workers (in case of failed deletions)
	for name, workerCapacityId := range workloadCapacityMap {
		if name == k.Master.Name {
			continue
		}
		workerWorkload := workloadObj[name]
		workerIP := workloadComputedIP[name]
		workerIP6 := workloadComputedIP6[name]
		workerDiskSize := workloadDiskSize[name]
		workerNodeId, err := getNodeIdByCapacityId(sub, workerCapacityId)
		if err != nil {
			errors.Wrapf(err, "couldn't get worker nodeID, capacityId: %d", workerCapacityId)
		}
		w, err := NewK8sNodeDataFromWorkload(workerWorkload, workerNodeId, workerCapacityId, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
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
	if k.PublicIP || k.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", k.Name)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName, k.PublicIP, k.PublicIP6))
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
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: k.Planetary,
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

func (k *K8sDeployer) getK8sFreeIP(ipRange gridtypes.IPNet, nodeID uint32) (string, error) {
	for i := byte(2); i <= byte(255); i++ {
		if !includes(k.NodeUsedIPs[nodeID], i) {
			k.NodeUsedIPs[nodeID] = append(k.NodeUsedIPs[nodeID], i)
			ip := ipRange.IP.To4()
			ip[3] = i
			return ip.String(), nil
		}
	}
	return "", errors.New("all ips are used")
}

func resourceK8sCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	err = deployer.Deploy(ctx, apiClient.substrateConn, d, apiClient)
	if err != nil {
		if len(deployer.CapacityDeploymentMap) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	deployer.storeState(d, apiClient)
	d.SetId(uuid.New().String())
	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	if err := deployer.invalidateBrokenAttributes(apiClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.Deploy(ctx, apiClient.substrateConn, d, apiClient)
	if err != nil {
		diags = diag.FromErr(err)
	}
	deployer.storeState(d, apiClient)
	return diags
}

func resourceK8sRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	if err := deployer.invalidateBrokenAttributes(apiClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.updateFromRemote(ctx, apiClient.substrateConn)
	log.Printf("read updateFromRemote err: %s\n", err)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   err.Error(),
		})
		return diags
	}
	deployer.storeState(d, apiClient)
	return diags
}

func resourceK8sDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewK8sDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	err = deployer.Cancel(ctx, apiClient.substrateConn, d, apiClient)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
	} else {
		deployer.storeState(d, apiClient)
	}
	return diags
}
