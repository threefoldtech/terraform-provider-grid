package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const ExternalNodeID = -1

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Network resource.",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network Name",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Project Name",
				Default:     "Network",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"capacity_contracts": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of capacity reservation contracts the network should be deployed on",
			},
			"nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of nodes to add to the network",
			},
			"ip_range": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network ip range",
			},
			"add_wg_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to add a public node to network and use it to generate a wg config",
			},
			"access_wg_config": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "WG config for access",
			},
			"external_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP of the access point (the IP to use in local wireguard config)",
			},
			"external_sk": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Access point private key (the one to use in the local wireguard config to access the network)",
			},
			"public_node_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Public node id (in case it's added). Used for wireguard access and supporting hidden nodes.",
			},
			"nodes_ip_range": {
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Computed values of nodes' ip ranges after deployment",
			},
			"node_deployment_id": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each node to its deployment id",
			},
		},
	}
}

type NetworkDeployer struct {
	Name              string
	Description       string
	CapacityContracts []uint64 `name:"capacity_contracts"`
	IPRange           gridtypes.IPNet
	AddWGAccess       bool

	Nodes            []uint32
	AccessWGConfig   string
	ExternalIP       *gridtypes.IPNet
	ExternalSK       wgtypes.Key
	PublicNodeID     uint32
	NodeDeploymentID map[uint32]uint64
	NodeCapacityID   map[uint32]uint64
	NodesIPRange     map[uint32]gridtypes.IPNet

	WGPort         map[uint32]int
	Keys           map[uint32]wgtypes.Key
	APIClient      *apiClient
	ncPool         *client.NodeClientPool
	deploymentData string
	deployer       deployer.MultiDeployerInterface
}

func NewNetworkDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (NetworkDeployer, error) {
	var err error
	// nodesIf := d.Get("nodes").([]interface{})
	// nodes := make([]uint32, len(nodesIf))
	// for idx, n := range nodesIf {
	// 	nodes[idx] = uint32(n.(int))
	// }

	capacityContractsIf := d.Get("capacity_reservation_contract_ids").([]interface{})
	capacityContracts := make([]uint64, len(capacityContractsIf))
	for idx, id := range capacityContractsIf {
		capacityContracts[idx] = id.(uint64)
	}

	nodeCapacityIDIf := d.Get("node_capacity_id").(map[string]interface{})
	nodeCapacityID := map[uint32]uint64{}
	for node, id := range nodeCapacityIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		nodeCapacityID[uint32(nodeInt)] = uint64(id.(int))
	}

	nodes, err := getUniqueNodes(capacityContracts, nodeCapacityID, apiClient)
	if err != nil {
		return NetworkDeployer{}, nil
	}

	nodeDeploymentIDIf := d.Get("node_deployment_id").(map[string]interface{})
	nodeDeploymentID := make(map[uint32]uint64)
	for node, id := range nodeDeploymentIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRangeIf := d.Get("nodes_ip_range").(map[string]interface{})
	for node, r := range nodesIPRangeIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse node id")
		}
		nodesIPRange[uint32(nodeInt)], err = gridtypes.ParseIPNet(r.(string))
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse node ip range")
		}
	}

	// external node related data
	addWGAccess := d.Get("add_wg_access").(bool)

	var externalIP *gridtypes.IPNet
	externalIPStr := d.Get("external_ip").(string)
	if externalIPStr != "" {
		ip, err := gridtypes.ParseIPNet(externalIPStr)
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse external ip")
		}
		externalIP = &ip
	}
	var externalSK wgtypes.Key
	if d.Get("external_sk").(string) != "" {
		externalSK, err = wgtypes.ParseKey(d.Get("external_sk").(string))
	} else {
		externalSK, err = wgtypes.GeneratePrivateKey()
	}
	if err != nil {
		return NetworkDeployer{}, errors.Wrap(err, "failed to get external_sk key")
	}

	ipRange, err := gridtypes.ParseIPNet(d.Get("ip_range").(string))
	if err != nil {
		return NetworkDeployer{}, errors.Wrap(err, "couldn't parse network ip range")
	}
	pool := client.NewNodeClientPool(apiClient.rmb)
	deploymentData := DeploymentData{
		Name:        d.Get("name").(string),
		Type:        "network",
		ProjectName: d.Get("solution_type").(string),
	}
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}
	multiDeployere := deployer.MultiDeployer{
		Single: deployer.SingleDeployer{},
	}
	deployer := NetworkDeployer{
		Name:              d.Get("name").(string),
		Description:       d.Get("description").(string),
		CapacityContracts: capacityContracts,
		Nodes:             nodes,
		IPRange:           ipRange,
		AddWGAccess:       addWGAccess,
		AccessWGConfig:    d.Get("access_wg_config").(string),
		ExternalIP:        externalIP,
		ExternalSK:        externalSK,
		PublicNodeID:      uint32(d.Get("public_node_id").(int)),
		NodesIPRange:      nodesIPRange,
		NodeDeploymentID:  nodeDeploymentID,
		NodeCapacityID:    nodeCapacityID,
		Keys:              make(map[uint32]wgtypes.Key),
		WGPort:            make(map[uint32]int),
		APIClient:         apiClient,
		ncPool:            pool,
		deploymentData:    string(deploymentDataStr),
		deployer:          &multiDeployere,
	}
	return deployer, nil
}

func getUniqueNodes(contractIDs []uint64, nodeCapacityID map[uint32]uint64, cl *apiClient) ([]uint32, error) {
	cp := contractIDs[:]
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	nodes := []uint32{}
	for _, id := range cp {
		contract, err := cl.substrateConn.GetContract(id)
		if err != nil {
			return nil, err
		}
		node := uint32(contract.ContractType.CapacityReservationContract.NodeID)
		if isInUint64(cp, nodeCapacityID[node]) {
			// this node is already set to another capacity contract that exists in contarctIDs, no need to reset it
			continue
		}
		if !isInUint32(nodes, node) {
			nodes = append(nodes, node)
			nodeCapacityID[node] = id
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i] < nodes[j] })
	return nodes, nil
}

// invalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *NetworkDeployer) invalidateBrokenAttributes(sub *substrate.Substrate) error {

	for node, deploymentID := range k.NodeDeploymentID {
		// this is invalid now, get contract gets capacity contract, not deployment
		contract, err := sub.GetDeployment(deploymentID)
		if (err == nil && !contract.State.IsCreated) || errors.Is(err, substrate.ErrNotFound) {
			delete(k.NodeDeploymentID, node)
			delete(k.NodesIPRange, node)
			delete(k.Keys, node)
			delete(k.WGPort, node)
		} else if err != nil {
			return errors.Wrapf(err, "couldn't get node %d contract %d", node, deploymentID)
		}
	}
	if k.ExternalIP != nil && !k.IPRange.Contains(k.ExternalIP.IP) {
		k.ExternalIP = nil
	}
	for node, ip := range k.NodesIPRange {
		if !k.IPRange.Contains(ip.IP) {
			delete(k.NodesIPRange, node)
		}
	}
	if k.PublicNodeID != 0 {
		// TODO: add a check that the node is still public
		cl, err := k.ncPool.GetNodeClient(sub, k.PublicNodeID)
		if err != nil {
			// whatever the error, delete it and it will get reassigned later
			k.PublicNodeID = 0
		}
		if err := isNodeUp(context.Background(), cl); err != nil {
			k.PublicNodeID = 0
		}
	}

	if !k.AddWGAccess {
		k.ExternalIP = nil
	}
	return nil
}
func (k *NetworkDeployer) Validate(ctx context.Context, sub *substrate.Substrate) error {
	if err := validateAccountMoneyForExtrinsics(sub, k.APIClient.identity); err != nil {
		return err
	}
	mask := k.IPRange.Mask
	if ones, _ := mask.Size(); ones != 16 {
		return fmt.Errorf("subnet in iprange %s should be 16", k.IPRange.String())
	}

	return isNodesUp(ctx, sub, k.Nodes, k.ncPool)
}

func (k *NetworkDeployer) ValidateDelete(ctx context.Context) error {
	return nil
}

func (k *NetworkDeployer) storeState(d *schema.ResourceData, state state.StateI) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range k.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	nodesIPRange := make(map[string]interface{})
	for node, r := range k.NodesIPRange {
		nodesIPRange[fmt.Sprintf("%d", node)] = r.String()
	}

	nodes := make([]uint32, 0)
	for _, node := range k.Nodes {
		if _, ok := k.NodeDeploymentID[node]; ok {
			nodes = append(nodes, node)
		}
	}
	for node := range k.NodeDeploymentID {
		if !isInUint32(nodes, node) {
			if k.PublicNodeID == node {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	log.Printf("setting deployer object nodes: %v\n", nodes)
	// update network local status
	k.updateNetworkLocalState(state)

	k.Nodes = nodes

	log.Printf("storing nodes: %v\n", nodes)
	d.Set("nodes", nodes)
	d.Set("ip_range", k.IPRange.String())
	d.Set("access_wg_config", k.AccessWGConfig)
	if k.ExternalIP == nil {
		d.Set("external_ip", nil)
	} else {

		d.Set("external_ip", k.ExternalIP.String())
	}
	d.Set("external_sk", k.ExternalSK.String())
	d.Set("public_node_id", k.PublicNodeID)
	// plural or singular?
	d.Set("nodes_ip_range", nodesIPRange)
	d.Set("node_deployment_id", nodeDeploymentID)
}

func (k *NetworkDeployer) updateNetworkLocalState(state state.StateI) {
	ns := state.GetNetworkState()
	ns.DeleteNetwork(k.Name)
	network := ns.GetNetwork(k.Name)
	for nodeID, subnet := range k.NodesIPRange {
		network.SetNodeSubnet(nodeID, subnet.String())
	}
}

func nextFreeOctet(used []byte, start *byte) error {
	for isInByte(used, *start) && *start <= 254 {
		*start += 1
	}
	if *start == 255 {
		return errors.New("couldn't find a free ip to add node")
	}
	return nil
}

func (k *NetworkDeployer) assignNodesIPs(nodes []uint32) error {
	ips := make(map[uint32]gridtypes.IPNet)
	l := len(k.IPRange.IP)
	usedIPs := make([]byte, 0) // the third octet
	for node, ip := range k.NodesIPRange {
		if isInUint32(nodes, node) {
			usedIPs = append(usedIPs, ip.IP[l-2])
			ips[node] = ip
		}
	}
	var cur byte = 2
	if k.AddWGAccess {
		if k.ExternalIP != nil {
			usedIPs = append(usedIPs, k.ExternalIP.IP[l-2])
		} else {
			err := nextFreeOctet(usedIPs, &cur)
			if err != nil {
				return err
			}
			usedIPs = append(usedIPs, cur)
			ip := ipNet(k.IPRange.IP[l-4], k.IPRange.IP[l-3], cur, k.IPRange.IP[l-1], 24)
			k.ExternalIP = &ip
		}
	}
	for _, node := range nodes {
		if _, ok := ips[node]; !ok {
			err := nextFreeOctet(usedIPs, &cur)
			if err != nil {
				return err
			}
			usedIPs = append(usedIPs, cur)
			ips[node] = ipNet(k.IPRange.IP[l-4], k.IPRange.IP[l-3], cur, k.IPRange.IP[l-2], 24)
		}
	}
	k.NodesIPRange = ips
	return nil
}
func (k *NetworkDeployer) assignNodesWGPort(ctx context.Context, sub *substrate.Substrate, nodes []uint32) error {
	for _, node := range nodes {
		if _, ok := k.WGPort[node]; !ok {
			cl, err := k.ncPool.GetNodeClient(sub, node)
			if err != nil {
				return errors.Wrap(err, "coudln't get node client")
			}
			port, err := getNodeFreeWGPort(ctx, cl, node)
			if err != nil {
				return errors.Wrap(err, "failed to get node free wg ports")
			}
			k.WGPort[node] = port
		}
	}

	return nil
}
func (k *NetworkDeployer) assignNodesWGKey(nodes []uint32) error {
	for _, node := range nodes {
		if _, ok := k.Keys[node]; !ok {

			key, err := wgtypes.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "failed to generate wg private key")
			}
			k.Keys[node] = key
		}
	}

	return nil
}
func (k *NetworkDeployer) readNodesConfig(ctx context.Context, sub *substrate.Substrate) error {
	keys := make(map[uint32]wgtypes.Key)
	WGPort := make(map[uint32]int)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	log.Printf("reading node config")
	nodeDeployments, err := k.deployer.GetDeploymentObjects(ctx, sub, k.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}
	printDeployments(nodeDeployments)
	WGAccess := false
	for node, dl := range nodeDeployments {
		for _, wl := range dl.Workloads {
			if wl.Type != zos.NetworkType {
				continue
			}
			data, err := wl.WorkloadData()
			if err != nil {
				return errors.Wrap(err, "couldn't parse workload data")
			}

			d := data.(*zos.Network)
			WGPort[node] = int(d.WGListenPort)
			keys[node], err = wgtypes.ParseKey(d.WGPrivateKey)
			if err != nil {
				return errors.Wrap(err, "couldn't parse wg private key from workload object")
			}
			nodesIPRange[node] = d.Subnet
			// this will fail when hidden node is supported
			for _, peer := range d.Peers {
				if peer.Endpoint == "" {
					WGAccess = true
				}
			}
		}
	}
	k.Keys = keys
	k.WGPort = WGPort
	k.NodesIPRange = nodesIPRange
	k.AddWGAccess = WGAccess
	if !WGAccess {
		k.AccessWGConfig = ""
	}
	return nil
}

func (k *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context, sub *substrate.Substrate) ([]deployer.DeploymentProps, error) {
	log.Printf("nodes: %v\n", k.Nodes)
	props := []deployer.DeploymentProps{}
	endpoints := make(map[uint32]string)
	hiddenNodes := make([]uint32, 0)
	var ipv4Node uint32
	accessibleNodes := make([]uint32, 0)
	for _, node := range k.Nodes {
		cl, err := k.ncPool.GetNodeClient(sub, node)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't get node %d client", node)
		}
		endpoint, err := getNodeEndpoint(ctx, cl)
		if errors.Is(err, ErrNoAccessibleInterfaceFound) {
			hiddenNodes = append(hiddenNodes, node)
		} else if err != nil {
			return nil, errors.Wrapf(err, "failed to get node %d endpoint", node)
		} else if endpoint.To4() != nil {
			accessibleNodes = append(accessibleNodes, node)
			ipv4Node = node
			endpoints[node] = endpoint.String()
		} else {
			accessibleNodes = append(accessibleNodes, node)
			endpoints[node] = fmt.Sprintf("[%s]", endpoint.String())
		}
	}
	needsIPv4Access := k.AddWGAccess || (len(hiddenNodes) != 0 && len(hiddenNodes)+len(accessibleNodes) > 1)
	if needsIPv4Access {
		if k.PublicNodeID != 0 { // it's set
			// if public node id is already set, it should be added to accessible nodes
			if !isInUint32(accessibleNodes, k.PublicNodeID) {
				accessibleNodes = append(accessibleNodes, k.PublicNodeID)
			}
		} else if ipv4Node != 0 { // there's one in the network original nodes
			k.PublicNodeID = ipv4Node
		} else {
			publicNode, err := getPublicNode(ctx, k.APIClient.grid_client, []uint32{})
			if err != nil {
				return nil, errors.Wrap(err, "public node needed because you requested adding wg access or a hidden node is added to the network")
			}
			k.PublicNodeID = publicNode
			accessibleNodes = append(accessibleNodes, publicNode)
		}
		if endpoints[k.PublicNodeID] == "" { // old or new outsider
			cl, err := k.ncPool.GetNodeClient(sub, k.PublicNodeID)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't get node %d client", k.PublicNodeID)
			}
			endpoint, err := getNodeEndpoint(ctx, cl)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get node %d endpoint", k.PublicNodeID)
			}
			endpoints[k.PublicNodeID] = endpoint.String()
		}
	}
	all := append(hiddenNodes, accessibleNodes...)
	if err := k.assignNodesIPs(all); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node ips")
	}
	if err := k.assignNodesWGKey(all); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node wg keys")
	}
	if err := k.assignNodesWGPort(ctx, sub, all); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node wg ports")
	}
	nonAccessibleIPRanges := []gridtypes.IPNet{}
	for _, node := range hiddenNodes {
		r := k.NodesIPRange[node]
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, wgIP(r))
	}
	if k.AddWGAccess {
		r := k.ExternalIP
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, *r)
		nonAccessibleIPRanges = append(nonAccessibleIPRanges, wgIP(*r))
	}
	log.Printf("hidden nodes: %v\n", hiddenNodes)
	log.Printf("public node: %v\n", k.PublicNodeID)
	log.Printf("accessible nodes: %v\n", accessibleNodes)
	log.Printf("non accessible ip ranges: %v\n", nonAccessibleIPRanges)

	if k.AddWGAccess {
		k.AccessWGConfig = generateWGConfig(
			wgIP(*k.ExternalIP).IP.String(),
			k.ExternalSK.String(),
			k.Keys[k.PublicNodeID].PublicKey().String(),
			fmt.Sprintf("%s:%d", endpoints[k.PublicNodeID], k.WGPort[k.PublicNodeID]),
			k.IPRange.String(),
		)
	}

	for _, node := range accessibleNodes {
		peers := make([]zos.Peer, 0, len(k.Nodes))
		for _, neigh := range accessibleNodes {
			if neigh == node {
				continue
			}
			neighIPRange := k.NodesIPRange[neigh]
			allowed_ips := []gridtypes.IPNet{
				neighIPRange,
				wgIP(neighIPRange),
			}
			if neigh == k.PublicNodeID {
				allowed_ips = append(allowed_ips, nonAccessibleIPRanges...)
			}
			peers = append(peers, zos.Peer{
				Subnet:      k.NodesIPRange[neigh],
				WGPublicKey: k.Keys[neigh].PublicKey().String(),
				Endpoint:    fmt.Sprintf("%s:%d", endpoints[neigh], k.WGPort[neigh]),
				AllowedIPs:  allowed_ips,
			})
		}
		if node == k.PublicNodeID {
			// external node
			if k.AddWGAccess {
				peers = append(peers, zos.Peer{
					Subnet:      *k.ExternalIP,
					WGPublicKey: k.ExternalSK.PublicKey().String(),
					AllowedIPs:  []gridtypes.IPNet{*k.ExternalIP, wgIP(*k.ExternalIP)},
				})
			}
			// hidden nodes
			for _, neigh := range hiddenNodes {
				neighIPRange := k.NodesIPRange[neigh]
				peers = append(peers, zos.Peer{
					Subnet:      neighIPRange,
					WGPublicKey: k.Keys[neigh].PublicKey().String(),
					AllowedIPs: []gridtypes.IPNet{
						neighIPRange,
						wgIP(neighIPRange),
					},
				})
			}
		}

		workload := gridtypes.Workload{
			Version:     0,
			Type:        zos.NetworkType,
			Description: k.Description,
			Name:        gridtypes.Name(k.Name),
			Data: gridtypes.MustMarshal(zos.Network{
				NetworkIPRange: gridtypes.MustParseIPNet(k.IPRange.String()),
				Subnet:         k.NodesIPRange[node],
				WGPrivateKey:   k.Keys[node].String(),
				WGListenPort:   uint16(k.WGPort[node]),
				Peers:          peers,
			}),
		}
		deployment := gridtypes.Deployment{
			Version: 0,
			TwinID:  k.APIClient.twin_id, //LocalTwin,
			// this contract id must match the one on substrate
			Workloads: []gridtypes.Workload{
				workload,
			},
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
		// deployments[node] = deployment
		props = append(props, deployer.DeploymentProps{
			ContractID: deployer.CapacityReservationContractID(k.NodeCapacityID[node]),
			Deployment: deployment,
		})
	}
	// hidden nodes deployments
	for _, node := range hiddenNodes {
		nodeIPRange := k.NodesIPRange[node]
		peers := make([]zos.Peer, 0)
		if k.PublicNodeID != 0 {
			peers = append(peers, zos.Peer{
				WGPublicKey: k.Keys[k.PublicNodeID].PublicKey().String(),
				Subnet:      nodeIPRange,
				AllowedIPs: []gridtypes.IPNet{
					k.IPRange,
					ipNet(100, 64, 0, 0, 16),
				},
				Endpoint: fmt.Sprintf("%s:%d", endpoints[k.PublicNodeID], k.WGPort[k.PublicNodeID]),
			})
		}
		workload := gridtypes.Workload{
			Version:     0,
			Type:        zos.NetworkType,
			Description: k.Description,
			Name:        gridtypes.Name(k.Name),
			Data: gridtypes.MustMarshal(zos.Network{
				NetworkIPRange: gridtypes.MustParseIPNet(k.IPRange.String()),
				Subnet:         nodeIPRange,
				WGPrivateKey:   k.Keys[node].String(),
				WGListenPort:   uint16(k.WGPort[node]),
				Peers:          peers,
			}),
		}
		deployment := gridtypes.Deployment{
			Version: 0,
			TwinID:  k.APIClient.twin_id,
			Workloads: []gridtypes.Workload{
				workload,
			},
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
		// deployments[node] = deployment
		props = append(props, deployer.DeploymentProps{
			ContractID: deployer.CapacityReservationContractID(k.NodeCapacityID[node]),
			Deployment: deployment,
		})
	}
	return props, nil
}

//	func (k *NetworkDeployer) Deploy(ctx context.Context, sub *substrate.Substrate) error {
//		newDeployments, err := k.GenerateVersionlessDeployments(ctx, sub)
//		if err != nil {
//			return errors.Wrap(err, "couldn't generate deployments data")
//		}
//		log.Printf("new deployments")
//		printDeployments(newDeployments)
//		currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
//		if err := k.updateState(ctx, sub, currentDeployments); err != nil {
//			log.Printf("error updating state: %s\n", err)
//		}
//		return err
//	}
func (k *NetworkDeployer) updateState(ctx context.Context, sub *substrate.Substrate, currentDeploymentIDs map[uint32]uint64) error {
	k.NodeDeploymentID = currentDeploymentIDs
	if err := k.readNodesConfig(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't read node's data")
	}

	return nil
}

// func (k *NetworkDeployer) Cancel(ctx context.Context, sub *substrate.Substrate) error {
// 	newDeployments := make(map[uint32]gridtypes.Deployment)

// 	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
// 	if err := k.updateState(ctx, sub, currentDeployments); err != nil {
// 		log.Printf("error updating state: %s\n", err)
// 	}
// 	return err
// }

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	k, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := k.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}
	cl := &deployer.Client{
		Identity:  apiClient.identity,
		Sub:       apiClient.substrateConn,
		Twin:      apiClient.twin_id,
		GridProxy: apiClient.grid_client,
		NCPool:    k.ncPool,
	}
	// err = k.CreateNeededCapacityContracts(ctx, apiClient.substrateConn)
	// if err != nil {
	// 	return diag.FromErr(errors.Wrap(err, "couldn't create capacity contracts for network resource"))
	// }

	props, err := k.GenerateVersionlessDeployments(ctx, apiClient.substrateConn)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't generate deployments"))
	}
	err = k.deployer.Create(ctx, cl, deployer.DeploymentData(k.deploymentData), props)
	if err != nil {
		if len(k.NodeDeploymentID) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	k.storeState(d, apiClient.state)
	d.SetId(uuid.New().String())
	return diags
}

// UpdateCapacityContracts creates and deletes needed capacity contracts depending on nodes and wg access
// func (k *NetworkDeployer) UpdateCapacityContracts(ctx context.Context, sub *substrate.Substrate) error {
// 	// create capacity contracts for nodes
// 	// if a user requires wg access, create a separate capacity contract for the access node
// 	//
// 	// check k.Nodes and k.NetworkCapacityContracts
// 	// if a node doesn't have a corresponding capacity contract, create it
// 	// if user wants wg access and k.AccessNodeCapacityContract == 0, create a new capacity contract
// 	nodesWithCapacities := []uint32{}
// 	toCreate := []uint32{}
// 	toDelete := []uint64{}

// 	for _, id := range k.NetworkCapacityContracts {
// 		contract, err := sub.GetContract(id)
// 		if err != nil {
// 			return err
// 		}
// 		node := uint32(contract.ContractType.CapacityReservationContract.NodeID)
// 		if !isInUint32(k.Nodes, node) {
// 			toDelete = append(toDelete, id)
// 		}
// 		nodesWithCapacities = append(nodesWithCapacities, node)
// 	}

// 	// delete contracts
// 	for _, id := range toDelete {
// 		err := sub.CancelContract(k.APIClient.identity, id)
// 		if err != nil {
// 			return err
// 		}
// 		k.findAndRemoveContract(id)
// 	}
// 	// update k.NetworkCapacityContracts accordingly

// 	for _, node := range k.Nodes {
// 		if !isInUint32(nodesWithCapacities, node) {
// 			toCreate = append(toCreate, node)
// 		}
// 	}

// 	// create contracts
// 	// update k.NetworkCapacityContracts accordingly

// 	// choosing an access node
// 	if k.AddWGAccess {
// 		// check if k.AccessNodeCapacity exists
// 		// if not: create a contract
// 		// if exists: ok
// 		if k.AccessNodeCapacityContract == 0 {

// 		}
// 	} else {
// 		// check if k.AccessNodeCapacity exists
// 		// if exists: delete it
// 		// if no, ok
// 	}

// }

// func (k *NetworkDeployer) findAndRemoveContract(id uint64) {
// 	for idx := range k.NetworkCapacityContracts {
// 		if k.NetworkCapacityContracts[idx] == id {
// 			k.NetworkCapacityContracts[idx] = k.NetworkCapacityContracts[len(k.NetworkCapacityContracts)-1]
// 			k.NetworkCapacityContracts = k.NetworkCapacityContracts[:len(k.NetworkCapacityContracts)-1]
// 			return
// 		}
// 	}
// }

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	k, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := k.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}
	if err := k.invalidateBrokenAttributes(apiClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}
	props, err := k.GenerateVersionlessDeployments(ctx, apiClient.substrateConn)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't generate deployments"))
	}
	// cl := deployer
	// deployer.deployer.Create()
	cl := &deployer.Client{
		Identity:  apiClient.identity,
		Sub:       apiClient.substrateConn,
		Twin:      apiClient.twin_id,
		NCPool:    k.ncPool,
		GridProxy: apiClient.grid_client,
	}

	if err := k.CancelRemovedDeployments(ctx, cl, props); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't cancel removed deployments"))
	}
	if err := k.CreateNewDeployments(ctx, cl, props); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't create new deployments"))
	}
	if err := k.UpdateOldDeployments(ctx, cl, props); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't update old deployments"))
	}

	// deployment ids should be set in props beffore updating
	// create, update, delete operations should be separated

	// err = k.Deploy(ctx, apiClient.substrateConn)
	// if err != nil {
	// 	diags = diag.FromErr(err)
	// }
	k.storeState(d, apiClient.state)
	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.invalidateBrokenAttributes(apiClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.readNodesConfig(ctx, apiClient.substrateConn)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   err.Error(),
		})
		return diags
	}
	deployer.storeState(d, apiClient.state)
	return diags
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	k, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	toDelete := []deployer.DeploymentID{}
	for _, deploymentID := range k.NodeDeploymentID {
		toDelete = append(toDelete, deployer.DeploymentID(deploymentID))
	}
	cl := &deployer.Client{
		Identity:  apiClient.identity,
		Sub:       apiClient.substrateConn,
		Twin:      apiClient.twin_id,
		NCPool:    k.ncPool,
		GridProxy: apiClient.grid_client,
	}
	err = k.deployer.Delete(ctx, cl, toDelete)
	if err != nil {
		diags = diag.FromErr(err)
	}
	// err = k.Cancel(ctx, apiClient.substrateConn)
	// if err != nil {
	// 	diags = diag.FromErr(err)
	// }
	if err == nil {
		d.SetId("")
		ns := apiClient.state.GetNetworkState()
		ns.DeleteNetwork(k.Name)
	} else {
		k.storeState(d, apiClient.state)
	}
	return diags
}

func (k *NetworkDeployer) CancelRemovedDeployments(ctx context.Context, cl *deployer.Client, props []deployer.DeploymentProps) error {
	// if a node is in k.NodeDeploymentIDs and not in k.Nodes, cancel this deployment
	toDelete := []deployer.DeploymentID{}
	for node, deploymentID := range k.NodeDeploymentID {
		if !isInUint32(k.Nodes, node) {
			toDelete = append(toDelete, deployer.DeploymentID(deploymentID))
		}
	}

	// TODO: return deleted deployments???

	deleted, err := k.deployer.Delete(ctx, cl, toDelete)
	for _, d := range deleted {
		// find d in k.nodeDeploymentIDs and delete it
	}
	if err != nil {
		return errors.Wrap(err, "couldn't cancel deployments")
	}
	return nil
}

func (k *NetworkDeployer) CreateNewDeployments(ctx context.Context, cl *deployer.Client, props []deployer.DeploymentProps) error {
	// if a node is not in k.NodeDeploymentIDs and in k.Nodes, create this deployment
	toCreate := []deployer.DeploymentProps{}
	for _, p := range props {
		contract, err := cl.Sub.GetContract(p.Deployment.DeploymentID.U64())
		if err != nil {
			return err
		}
		node := contract.ContractType.CapacityReservationContract.NodeID
		if _, ok := k.NodeDeploymentID[uint32(node)]; !ok {
			toCreate = append(toCreate, p)
		}
	}
	created, err := k.deployer.Create(ctx, cl, toCreate)
	for node, deploymentID := range created {
		k.NodeDeploymentID[node] = deploymentID
	}
	if err != nil {
		return errors.Wrap(err, "couldn't create deployments")
	}

}

func (k *NetworkDeployer) UpdateOldDeployments(ctx context.Context, cl *deployer.Client, props []deployer.DeploymentProps) error {
	// if a node is in k.NodeDeploymentIDs and in k.Nodes, there are two paths to take:
	// check if k.NodeCapacityID is in k.CapacityContractIDs:
	// 		if exists: update this deployment
	//		if not: cancel this deployment, recreate it.
	//
}
