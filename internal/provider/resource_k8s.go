// Package provider is the terraform provider
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
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
			"node_deployment_id": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each node to its deployment id",
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
							Required:    true,
							Description: "Node ID",
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
							Required:    true,
							Description: "Node ID",
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

type K8sDeployer struct {
	K8sCluster       workloads.K8sCluster
	NodesIPRange     map[uint32]gridtypes.IPNet
	NodeDeploymentID map[uint32]uint64

	ThreefoldPluginClient *threefoldPluginClient

	NodeUsedIPs map[uint32][]byte
	ncPool      *client.NodeClientPool
	d           *schema.ResourceData
	deployer    deployer.Deployer
}

func NewK8sDeployer(d *schema.ResourceData, threefoldPluginClient *threefoldPluginClient) (K8sDeployer, error) {
	networkName := d.Get("network_name").(string)
	ns := threefoldPluginClient.state.GetState().Networks
	network := ns.GetNetwork(networkName)

	master := workloads.NewK8sNodeData(d.Get("master").([]interface{})[0].(map[string]interface{}))
	workers := make([]workloads.K8sNodeData, 0)
	usedIPs := make(map[uint32][]byte)

	if master.IP != "" {
		usedIPs[master.Node] = append(usedIPs[master.Node], net.ParseIP(master.IP)[3])
	}
	usedIPs[master.Node] = append(usedIPs[master.Node], network.GetUsedNetworkHostIDs(master.Node)...)
	for _, w := range d.Get("workers").([]interface{}) {
		data := workloads.NewK8sNodeData(w.(map[string]interface{}))
		workers = append(workers, data)
		if data.IP != "" {
			usedIPs[data.Node] = append(usedIPs[data.Node], net.ParseIP(data.IP)[3])
			usedIPs[data.Node] = append(usedIPs[data.Node], network.GetUsedNetworkHostIDs(data.Node)...)
		}
	}
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	var err error
	nodesIPRange[master.Node], err = gridtypes.ParseIPNet(network.GetNodeSubnet(master.Node))
	if err != nil {
		return K8sDeployer{}, errors.Wrap(err, "couldn't parse master node ip range")
	}
	for _, worker := range workers {
		nodesIPRange[worker.Node], err = gridtypes.ParseIPNet(network.GetNodeSubnet(worker.Node))
		if err != nil {
			return K8sDeployer{}, errors.Wrapf(err, "couldn't parse worker node (%d) ip range", worker.Node)
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

	pool := client.NewNodeClientPool(threefoldPluginClient.rmb)
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
		K8sCluster: workloads.K8sCluster{
			Master:      &master,
			Workers:     workers,
			Token:       d.Get("token").(string),
			SSHKey:      d.Get("ssh_key").(string),
			NetworkName: d.Get("network_name").(string),
		},
		NodeDeploymentID:      nodeDeploymentID,
		NodeUsedIPs:           usedIPs,
		NodesIPRange:          nodesIPRange,
		ThreefoldPluginClient: threefoldPluginClient,
		ncPool:                pool,
		d:                     d,
		deployer:              deployer.NewDeployer(threefoldPluginClient.identity, threefoldPluginClient.twinID, threefoldPluginClient.gridProxyClient, pool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

// invalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *K8sDeployer) invalidateBrokenAttributes(sub subi.SubstrateExt) error {
	newWorkers := make([]workloads.K8sNodeData, 0)
	validNodes := make(map[uint32]struct{})
	for node, contractID := range k.NodeDeploymentID {
		contract, err := sub.GetContract(contractID)
		if (err == nil && !contract.IsCreated()) || errors.Is(err, subi.ErrNotFound) {
			delete(k.NodeDeploymentID, node)
			delete(k.NodesIPRange, node)
		} else if err != nil {
			return errors.Wrapf(err, "couldn't get node %d contract %d", node, contractID)
		} else {
			validNodes[node] = struct{}{}
		}

	}
	if _, ok := validNodes[k.K8sCluster.Master.Node]; !ok {
		k.K8sCluster.Master = &workloads.K8sNodeData{}
	}
	for _, worker := range k.K8sCluster.Workers {
		if _, ok := validNodes[worker.Node]; ok {
			newWorkers = append(newWorkers, worker)
		}
	}
	k.K8sCluster.Workers = newWorkers
	return nil
}

func (d *K8sDeployer) retainChecksums(workers []interface{}, master interface{}) {
	checksumMap := make(map[string]string)
	checksumMap[d.K8sCluster.Master.Name] = d.K8sCluster.Master.FlistChecksum
	for _, w := range d.K8sCluster.Workers {
		checksumMap[w.Name] = w.FlistChecksum
	}
	typed := master.(map[string]interface{})
	typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	for _, w := range workers {
		typed := w.(map[string]interface{})
		typed["flist_checksum"] = checksumMap[typed["name"].(string)]
	}
}

func (k *K8sDeployer) storeState(d *schema.ResourceData, cl *threefoldPluginClient) (errors error) {
	workers := make([]interface{}, 0)
	for _, w := range k.K8sCluster.Workers {
		workers = append(workers, w.Dictify())
	}
	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}
	log.Printf("master data: %v\n", k.K8sCluster.Master)
	if k.K8sCluster.Master == nil {
		k.K8sCluster.Master = &workloads.K8sNodeData{}
	}
	master := k.K8sCluster.Master.Dictify()
	k.retainChecksums(workers, master)

	l := []interface{}{master}
	k.updateNetworkState(d, cl.state)
	err := d.Set("master", l)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("workers", workers)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("token", k.K8sCluster.Token)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("ssh_key", k.K8sCluster.SSHKey)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("network_name", k.K8sCluster.NetworkName)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	return
}

func (k *K8sDeployer) updateNetworkState(d *schema.ResourceData, state state.StateGetter) {
	ns := state.GetState().Networks
	network := ns.GetNetwork(k.K8sCluster.NetworkName)
	before, _ := d.GetChange("node_deployment_id")
	for node, deploymentID := range before.(map[string]interface{}) {
		nodeID, err := strconv.Atoi(node)
		if err != nil {
			log.Printf("error converting node id string to int: %+v", err)
			continue
		}
		deploymentIDStr := fmt.Sprint(deploymentID.(int))
		network.DeleteDeploymentHostIDs(uint32(nodeID), deploymentIDStr)
	}
	// remove old ips
	network.DeleteDeploymentHostIDs(k.K8sCluster.Master.Node, fmt.Sprint(k.NodeDeploymentID[k.K8sCluster.Master.Node]))
	for _, worker := range k.K8sCluster.Workers {
		network.DeleteDeploymentHostIDs(worker.Node, fmt.Sprint(k.NodeDeploymentID[worker.Node]))
	}

	// append new ips
	masterNodeDeploymentHostIDs := network.GetDeploymentHostIDs(k.K8sCluster.Master.Node, fmt.Sprint(k.NodeDeploymentID[k.K8sCluster.Master.Node]))
	masterIP := net.ParseIP(k.K8sCluster.Master.IP)
	if masterIP == nil {
		log.Printf("couldn't parse master ip")
	} else {
		masterNodeDeploymentHostIDs = append(masterNodeDeploymentHostIDs, masterIP.To4()[3])
	}
	network.SetDeploymentHostIDs(k.K8sCluster.Master.Node, fmt.Sprint(k.NodeDeploymentID[k.K8sCluster.Master.Node]), masterNodeDeploymentHostIDs)
	for _, worker := range k.K8sCluster.Workers {
		workerNodeDeploymentHostIDs := network.GetDeploymentHostIDs(worker.Node, fmt.Sprint(k.NodeDeploymentID[worker.Node]))
		workerIP := net.ParseIP(worker.IP)
		if workerIP == nil {
			log.Printf("couldn't parse worker ip at node (%d)", worker.Node)
		} else {
			workerNodeDeploymentHostIDs = append(workerNodeDeploymentHostIDs, workerIP.To4()[3])
		}
		network.SetDeploymentHostIDs(worker.Node, fmt.Sprint(k.NodeDeploymentID[worker.Node]), workerNodeDeploymentHostIDs)
	}
}

func (k *K8sDeployer) assignNodesHostIDs() error {
	// TODO: when a k8s node changes its zos node, remove its ip from the used ones. better at the beginning
	masterNodeRange := k.NodesIPRange[k.K8sCluster.Master.Node]
	if k.K8sCluster.Master.IP == "" || !masterNodeRange.Contains(net.ParseIP(k.K8sCluster.Master.IP)) {
		ip, err := k.getK8sFreeIP(masterNodeRange, k.K8sCluster.Master.Node)
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for master")
		}
		k.K8sCluster.Master.IP = ip
	}
	for idx, w := range k.K8sCluster.Workers {
		workerNodeRange := k.NodesIPRange[w.Node]
		if w.IP != "" && workerNodeRange.Contains(net.ParseIP(w.IP)) {
			continue
		}
		ip, err := k.getK8sFreeIP(workerNodeRange, w.Node)
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for worker")
		}
		k.K8sCluster.Workers[idx].IP = ip
	}
	return nil
}
func (k *K8sDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	err := k.assignNodesHostIDs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	deployments := make(map[uint32]gridtypes.Deployment)
	nodeWorkloads := make(map[uint32][]gridtypes.Workload)
	masterWorkloads := k.K8sCluster.Master.GenerateK8sWorkload(&k.K8sCluster, "")
	nodeWorkloads[k.K8sCluster.Master.Node] = append(nodeWorkloads[k.K8sCluster.Master.Node], masterWorkloads...)
	for _, w := range k.K8sCluster.Workers {
		workerWorkloads := w.GenerateK8sWorkload(&k.K8sCluster, k.K8sCluster.Master.IP)
		nodeWorkloads[w.Node] = append(nodeWorkloads[w.Node], workerWorkloads...)
	}

	for node, ws := range nodeWorkloads {
		dl := gridtypes.Deployment{
			Version: 0,
			TwinID:  uint32(k.ThreefoldPluginClient.twinID), //LocalTwin,
			// this contract id must match the one on substrate
			Workloads: ws,
			SignatureRequirement: gridtypes.SignatureRequirement{
				WeightRequired: 1,
				Requests: []gridtypes.SignatureRequest{
					{
						TwinID: k.ThreefoldPluginClient.twinID,
						Weight: 1,
					},
				},
			},
		}
		deployments[node] = dl
	}
	return deployments, nil
}

func (d *K8sDeployer) validateChecksums() error {
	nodes := append(d.K8sCluster.Workers, *d.K8sCluster.Master)
	for _, vm := range nodes {
		if vm.FlistChecksum == "" {
			continue
		}
		checksum, err := workloads.GetFlistChecksum(vm.Flist)
		if err != nil {
			return errors.Wrapf(err, "couldn't get flist %s hash", vm.Flist)
		}
		if vm.FlistChecksum != checksum {
			return fmt.Errorf("passed checksum %s of %s doesn't match %s returned from %s",
				vm.FlistChecksum,
				vm.Name,
				checksum,
				workloads.FlistChecksumURL(vm.Flist),
			)
		}
	}
	return nil
}

func (k *K8sDeployer) ValidateIPranges(ctx context.Context) error {

	if _, ok := k.NodesIPRange[k.K8sCluster.Master.Node]; !ok {
		return fmt.Errorf("the master node %d doesn't exist in the network's ip ranges", k.K8sCluster.Master.Node)
	}
	for _, w := range k.K8sCluster.Workers {
		if _, ok := k.NodesIPRange[w.Node]; !ok {
			return fmt.Errorf("the node with id %d in worker %s doesn't exist in the network's ip ranges", w.Node, w.Name)
		}
	}
	return nil
}

func (k *K8sDeployer) Validate(ctx context.Context, sub subi.SubstrateExt) error {
	if err := k.K8sCluster.ValidateToken(ctx); err != nil {
		return err
	}
	if err := validateAccountBalanceForExtrinsics(sub, k.ThreefoldPluginClient.identity); err != nil {
		return err
	}
	if err := k.K8sCluster.ValidateNames(ctx); err != nil {
		return err
	}
	if err := k.ValidateIPranges(ctx); err != nil {
		return err
	}
	nodes := make([]uint32, 0)
	nodes = append(nodes, k.K8sCluster.Master.Node)
	for _, w := range k.K8sCluster.Workers {
		nodes = append(nodes, w.Node)

	}
	return client.AreNodesUp(ctx, sub, nodes, k.ncPool)
}

func (k *K8sDeployer) Deploy(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, cl *threefoldPluginClient) error {
	if err := k.validateChecksums(); err != nil {
		return err
	}
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	if err := k.updateState(ctx, sub, currentDeployments, d, cl); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func (k *K8sDeployer) Cancel(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, cl *threefoldPluginClient) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)

	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
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

func printDeployments(dls map[uint32]gridtypes.Deployment) (err error) {
	for node, dl := range dls {
		log.Printf("node id: %d\n", node)
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		err := enc.Encode(dl)
		if err != nil {
			return err
		}
	}

	return
}

func (k *K8sDeployer) removeUsedIPsFromLocalState(cl *threefoldPluginClient) {
	ns := cl.state.GetState().Networks
	network := ns.GetNetwork(k.K8sCluster.NetworkName)

	network.DeleteDeploymentHostIDs(k.K8sCluster.Master.Node, fmt.Sprint(k.NodeDeploymentID[k.K8sCluster.Master.Node]))
	for _, worker := range k.K8sCluster.Workers {
		network.DeleteDeploymentHostIDs(worker.Node, fmt.Sprint(k.NodeDeploymentID[worker.Node]))
	}
}

func (k *K8sDeployer) updateState(ctx context.Context, sub subi.SubstrateExt, currentDeploymentIDs map[uint32]uint64, d *schema.ResourceData, cl *threefoldPluginClient) error {
	log.Printf("current deployments\n")
	k.NodeDeploymentID = currentDeploymentIDs
	currentDeployments, err := k.deployer.GetDeployments(ctx, sub, currentDeploymentIDs)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}

	err = printDeployments(currentDeployments)
	if err != nil {
		return errors.Wrap(err, "couldn't print deployments data")
	}

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
	masterIPName := fmt.Sprintf("%sip", k.K8sCluster.Master.Name)
	k.K8sCluster.Master.ComputedIP = publicIPs[masterIPName]
	k.K8sCluster.Master.ComputedIP6 = publicIP6s[masterIPName]
	k.K8sCluster.Master.IP = privateIPs[string(k.K8sCluster.Master.Name)]
	k.K8sCluster.Master.YggIP = yggIPs[string(k.K8sCluster.Master.Name)]

	for idx, w := range k.K8sCluster.Workers {
		workerIPName := fmt.Sprintf("%sip", w.Name)
		k.K8sCluster.Workers[idx].ComputedIP = publicIPs[workerIPName]
		k.K8sCluster.Workers[idx].ComputedIP = publicIP6s[workerIPName]
		k.K8sCluster.Workers[idx].IP = privateIPs[string(w.Name)]
		k.K8sCluster.Workers[idx].YggIP = yggIPs[string(w.Name)]
	}
	k.updateNetworkState(d, cl.state)
	log.Printf("Current state after updatestate %v\n", k)
	return nil
}

func (k *K8sDeployer) removeDeletedContracts(ctx context.Context, sub subi.SubstrateExt) error {
	nodeDeploymentID := make(map[uint32]uint64)
	for nodeID, deploymentID := range k.NodeDeploymentID {
		cont, err := sub.GetContract(deploymentID)
		if err != nil {
			return errors.Wrap(err, "failed to get deployments")
		}
		if !cont.IsDeleted() {
			nodeDeploymentID[nodeID] = deploymentID
		}
	}
	k.NodeDeploymentID = nodeDeploymentID
	return nil
}

func (k *K8sDeployer) updateFromRemote(ctx context.Context, sub subi.SubstrateExt) error {
	if err := k.removeDeletedContracts(ctx, sub); err != nil {
		return errors.Wrap(err, "failed to remove deleted contracts")
	}
	currentDeployments, err := k.deployer.GetDeployments(ctx, sub, k.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch remote deployments")
	}
	log.Printf("calling updateFromRemote")
	err = printDeployments(currentDeployments)
	if err != nil {
		return errors.Wrap(err, "couldn't print deployments data")
	}

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
				if !keyUpdated && SSHKey != k.K8sCluster.SSHKey {
					k.K8sCluster.SSHKey = SSHKey
					keyUpdated = true
				}
				if !tokenUpdated && token != k.K8sCluster.Token {
					k.K8sCluster.Token = token
					tokenUpdated = true
				}
				if !networkUpdated && networkName != k.K8sCluster.NetworkName {
					k.K8sCluster.NetworkName = networkName
					networkUpdated = true
				}
			}
		}
	}

	nodeDeploymentID := make(map[uint32]uint64)
	for node, dl := range currentDeployments {
		nodeDeploymentID[node] = dl.ContractID
	}
	k.NodeDeploymentID = nodeDeploymentID
	// maps from workload name to (public ip, node id, disk size, actual workload)
	workloadNodeID := make(map[string]uint32)
	workloadDiskSize := make(map[string]int)
	workloadComputedIP := make(map[string]string)
	workloadComputedIP6 := make(map[string]string)
	workloadObj := make(map[string]gridtypes.Workload)

	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	diskSize := make(map[string]int)
	for node, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.ZMachineType {
				workloadNodeID[string(w.Name)] = node
				workloadObj[string(w.Name)] = w

			} else if w.Type == zos.PublicIPType {
				d := zos.PublicIPResult{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("failed to load pubip data %s", err)
					continue
				}
				publicIPs[string(w.Name)] = d.IP.String()
				publicIP6s[string(w.Name)] = d.IPv6.String()
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
	masterNodeID, ok := workloadNodeID[k.K8sCluster.Master.Name]
	if !ok {
		k.K8sCluster.Master = nil
	} else {
		masterWorkload := workloadObj[k.K8sCluster.Master.Name]
		masterIP := workloadComputedIP[k.K8sCluster.Master.Name]
		masterIP6 := workloadComputedIP6[k.K8sCluster.Master.Name]
		masterDiskSize := workloadDiskSize[k.K8sCluster.Master.Name]

		m, err := workloads.NewK8sNodeDataFromWorkload(masterWorkload, masterNodeID, masterDiskSize, masterIP, masterIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get master data from workload")
		}
		k.K8sCluster.Master = &m
	}
	// update workers
	workers := make([]workloads.K8sNodeData, 0)
	for _, w := range k.K8sCluster.Workers {
		workerNodeID, ok := workloadNodeID[w.Name]
		if !ok {
			// worker doesn't exist in any deployment, skip it
			continue
		}
		delete(workloadNodeID, w.Name)
		workerWorkload := workloadObj[w.Name]
		workerIP := workloadComputedIP[w.Name]
		workerIP6 := workloadComputedIP6[w.Name]

		workerDiskSize := workloadDiskSize[w.Name]
		w, err := workloads.NewK8sNodeDataFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
	// add missing workers (in case of failed deletions)
	for name, workerNodeID := range workloadNodeID {
		if name == k.K8sCluster.Master.Name {
			continue
		}
		workerWorkload := workloadObj[name]
		workerIP := workloadComputedIP[name]
		workerIP6 := workloadComputedIP6[name]
		workerDiskSize := workloadDiskSize[name]
		w, err := workloads.NewK8sNodeDataFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return errors.Wrap(err, "failed to get worker data from workload")
		}
		workers = append(workers, w)
	}
	k.K8sCluster.Workers = workers
	log.Printf("after updateFromRemote\n")
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	err = enc.Encode(k)
	if err != nil {
		return errors.Wrap(err, "failed to encode k8s deployer")
	}

	return nil
}

func (k *K8sDeployer) getK8sFreeIP(ipRange gridtypes.IPNet, nodeID uint32) (string, error) {
	ip := ipRange.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("the provided ip range (%s) is not a valid ipv4", ipRange.String())
	}

	for i := 2; i < 255; i++ {
		hostID := byte(i)
		if !Contains(k.NodeUsedIPs[nodeID], hostID) {
			k.NodeUsedIPs[nodeID] = append(k.NodeUsedIPs[nodeID], hostID)
			ip[3] = hostID
			return ip.String(), nil
		}
	}
	return "", errors.New("all ips are used")
}

func resourceK8sCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	threefoldPluginClient, ok := meta.(*threefoldPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	deployer, err := NewK8sDeployer(d, threefoldPluginClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, threefoldPluginClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	err = deployer.Deploy(ctx, threefoldPluginClient.substrateConn, d, threefoldPluginClient)
	if err != nil {
		if len(deployer.NodeDeploymentID) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	err = deployer.storeState(d, threefoldPluginClient)
	if err != nil {
		diags = diag.FromErr(err)
	}

	d.SetId(uuid.New().String())
	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	threefoldPluginClient, ok := meta.(*threefoldPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	deployer, err := NewK8sDeployer(d, threefoldPluginClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, threefoldPluginClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	if err := deployer.invalidateBrokenAttributes(threefoldPluginClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.Deploy(ctx, threefoldPluginClient.substrateConn, d, threefoldPluginClient)
	if err != nil {
		diags = diag.FromErr(err)
	}
	err = deployer.storeState(d, threefoldPluginClient)
	if err != nil {
		diags = diag.FromErr(err)
	}

	return diags
}

func resourceK8sRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	threefoldPluginClient, ok := meta.(*threefoldPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	deployer, err := NewK8sDeployer(d, threefoldPluginClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, threefoldPluginClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}

	if err := deployer.invalidateBrokenAttributes(threefoldPluginClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.updateFromRemote(ctx, threefoldPluginClient.substrateConn)
	log.Printf("read updateFromRemote err: %s\n", err)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  errTerraformOutSync,
			Detail:   err.Error(),
		})
		return diags
	}
	err = deployer.storeState(d, threefoldPluginClient)
	if err != nil {
		diags = diag.FromErr(err)
	}

	return diags
}

func resourceK8sDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	threefoldPluginClient, ok := meta.(*threefoldPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	deployer, err := NewK8sDeployer(d, threefoldPluginClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	err = deployer.Cancel(ctx, threefoldPluginClient.substrateConn, d, threefoldPluginClient)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
	} else {
		err = deployer.storeState(d, threefoldPluginClient)
		if err != nil {
			diags = diag.FromErr(err)
		}
	}
	return diags
}
