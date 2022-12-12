package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const ExternalNodeID = -1

const (
	NetworkSchemaName                  = "name"
	NetworkSchemaSolutionType          = "solution_type"
	NetworkSchemaDescription           = "description"
	NetworkSchemaCapacityIDs           = "capacity_ids"
	NetworkSchemaCapacityDeploymentMap = "capacity_deployment_map"
	NetworkSchemaAccessNodeCapacityID  = "access_node_capacity_id"
	NetworkSchemaNodeIDs               = "node_ids"
	NetworkSchemaNodeCapacityMap       = "node_capacity_map"
	NetworkSchemaIPRange               = "ip_range"
	NetworkSchemaAddWGAccess           = "add_wg_access"
	NetworkSchemaAccessWGConfig        = "access_wg_config"
	NetworkSchemaExternalIP            = "external_ip"
	NetworkSchemaExternalSK            = "external_sk"
	NetworkSchemaPublicNodeID          = "public_node_id"
	NetworkSchemaNodesIPRange          = "ndoes_ip_range"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Network resource.",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			NetworkSchemaName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network Name",
			},
			NetworkSchemaSolutionType: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Project Name",
				Default:     "Network",
			},
			NetworkSchemaDescription: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			NetworkSchemaCapacityIDs: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of capacity contract ids that the user wants to deploy network on",
			},
			NetworkSchemaCapacityDeploymentMap: {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "Map from capactiy contract id to deployment id",
			},
			NetworkSchemaAccessNodeCapacityID: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Access node capacity contract id that is created if an access node is required (user requires wg access, or all nodes don't a have public configuration).",
			},
			NetworkSchemaNodeIDs: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of nodes to add to the network",
			},
			NetworkSchemaNodeCapacityMap: {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "Map from node id to it's capacity contract id.",
			},
			NetworkSchemaIPRange: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network ip range",
			},
			NetworkSchemaAddWGAccess: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to add a public node to network and use it to generate a wg config",
			},
			NetworkSchemaAccessWGConfig: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "WG config for access",
			},
			NetworkSchemaExternalIP: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP of the access point (the IP to use in local wireguard config)",
			},
			NetworkSchemaExternalSK: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Access point private key (the one to use in the local wireguard config to access the network)",
			},
			NetworkSchemaPublicNodeID: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Public node id (in case it's added). Used for wireguard access and supporting hidden nodes.",
			},
			NetworkSchemaNodesIPRange: {
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Computed values of nodes' ip ranges after deployment",
			},
		},
	}
}

type NetworkDeployer struct {
	Name        string
	Description string
	CapacityIDs []uint64
	IPRange     gridtypes.IPNet
	AddWGAccess bool

	NodeIDs               []uint32
	AccessWGConfig        string
	AccessNodeCapacityID  uint64
	ExternalIP            *gridtypes.IPNet
	ExternalSK            wgtypes.Key
	PublicNodeID          uint32
	NodesIPRange          map[uint32]gridtypes.IPNet
	CapacityDeploymentMap map[uint64]uint64
	NodeCapacityMap       map[uint32]uint64

	CapacityNode map[uint64]uint32
	WGPort       map[uint32]int
	Keys         map[uint32]wgtypes.Key
	APIClient    *apiClient
	ncPool       *client.NodeClientPool
	deployer     deployer.Deployer
}

func NewNetworkDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (NetworkDeployer, error) {
	var err error
	capacityIDsIf := d.Get(NetworkSchemaCapacityIDs).([]interface{})
	capacityIDs := make([]uint64, len(capacityIDsIf))
	for idx, c := range capacityIDsIf {
		capacityIDs[idx] = uint64(c.(int))
	}

	capacityDeploymentMapIf := d.Get(NetworkSchemaCapacityDeploymentMap).(map[string]interface{})
	capacityDeploymentMap := map[uint64]uint64{}
	for capacityStr, deploymentID := range capacityDeploymentMapIf {
		capacityID, err := strconv.ParseUint(capacityStr, 10, 64)
		if err != nil {
			return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse capacity id %s", capacityStr)
		}
		capacityDeploymentMap[capacityID] = uint64(deploymentID.(int))
	}

	nodeCapacityMapIf := d.Get(NetworkSchemaNodeCapacityMap).(map[string]interface{})
	nodeCapacityMap := map[uint32]uint64{}
	for node, id := range nodeCapacityMapIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse node id %s", node)
		}
		nodeCapacityMap[uint32(nodeInt)] = uint64(id.(int))
	}

	nodeCapacityMap, err = assignNodesToCapacities(capacityIDs, nodeCapacityMap, apiClient)
	if err != nil {
		return NetworkDeployer{}, fmt.Errorf("couldn't assign nodes to capacities. %w", err)
	}

	capacityNode := map[uint64]uint32{}
	nodes := []uint32{}
	for node, capacity := range nodeCapacityMap {
		nodes = append(nodes, node)
		capacityNode[capacity] = node
	}

	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRangeIf := d.Get(NetworkSchemaNodesIPRange).(map[string]interface{})
	for node, r := range nodesIPRangeIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse node id %s", node)
		}
		nodesIPRange[uint32(nodeInt)], err = gridtypes.ParseIPNet(r.(string))
		if err != nil {
			return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse node ip range %s", r.(string))
		}
	}

	// external node related data
	addWGAccess := d.Get(NetworkSchemaAddWGAccess).(bool)

	var externalIP *gridtypes.IPNet
	externalIPStr := d.Get(NetworkSchemaExternalIP).(string)
	if externalIPStr != "" {
		ip, err := gridtypes.ParseIPNet(externalIPStr)
		if err != nil {
			return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse external ip %s", externalIPStr)
		}
		externalIP = &ip
	}
	var externalSK wgtypes.Key
	externalSKStr := d.Get(NetworkSchemaExternalSK).(string)
	if externalSKStr != "" {
		externalSK, err = wgtypes.ParseKey(externalSKStr)
	} else {
		externalSK, err = wgtypes.GeneratePrivateKey()
	}
	if err != nil {
		return NetworkDeployer{}, errors.Wrapf(err, "failed to get external_sk key %s", externalSKStr)
	}
	ipRangeStr := d.Get(NetworkSchemaIPRange).(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	if err != nil {
		return NetworkDeployer{}, errors.Wrapf(err, "couldn't parse network ip range %s", ipRangeStr)
	}
	pool := client.NewNodeClientPool(apiClient.rmb)
	deploymentData := DeploymentData{
		Name:        d.Get(NetworkSchemaName).(string),
		Type:        "network",
		ProjectName: d.Get(NetworkSchemaSolutionType).(string),
	}
	deploymentDataStr, err := json.Marshal(deploymentData)
	if err != nil {
		log.Printf("error parsing deploymentdata: %s", err.Error())
	}
	deployer := NetworkDeployer{
		Name:                  d.Get(NetworkSchemaName).(string),
		Description:           d.Get(NetworkSchemaDescription).(string),
		CapacityIDs:           capacityIDs,
		NodeIDs:               nodes,
		IPRange:               ipRange,
		AddWGAccess:           addWGAccess,
		AccessNodeCapacityID:  d.Get(NetworkSchemaAccessNodeCapacityID).(uint64),
		AccessWGConfig:        d.Get(NetworkSchemaAccessWGConfig).(string),
		ExternalIP:            externalIP,
		ExternalSK:            externalSK,
		PublicNodeID:          uint32(d.Get(NetworkSchemaPublicNodeID).(int)),
		NodesIPRange:          nodesIPRange,
		CapacityDeploymentMap: capacityDeploymentMap,
		NodeCapacityMap:       nodeCapacityMap,
		CapacityNode:          capacityNode,
		Keys:                  make(map[uint32]wgtypes.Key),
		WGPort:                make(map[uint32]int),
		APIClient:             apiClient,
		ncPool:                pool,
		deployer:              deployer.NewDeployer(apiClient.identity, apiClient.twin_id, apiClient.grid_client, pool, true, nil, string(deploymentDataStr)),
	}
	return deployer, nil
}

// assignNodesToCapacities should assing each node to one capacity contract id
func assignNodesToCapacities(contractIDs []uint64, nodeCapacityID map[uint32]uint64, cl *apiClient) (map[uint32]uint64, error) {
	newNodeCapacityID := map[uint32]uint64{}
	for _, id := range contractIDs {
		contract, err := cl.substrateConn.GetContract(id)
		if err != nil {
			return nil, fmt.Errorf("couldn't get capacity contract with id %d. %w", id, err)
		}
		node := uint32(contract.ContractType.CapacityReservationContract.NodeID)
		if includes[uint64](contractIDs, nodeCapacityID[node]) {
			// this node is already assigned to another capacity contract that exists in contarctIDs, no need to reset it
			continue
		}
		newNodeCapacityID[node] = id
	}
	return newNodeCapacityID, nil
}

// invalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *NetworkDeployer) invalidateBrokenAttributes(sub *substrate.Substrate) error {

	for capacity, deploymentID := range k.CapacityDeploymentMap {
		contract, err := sub.GetContract(capacity)
		if (err == nil && !contract.State.IsCreated) || errors.Is(err, substrate.ErrNotFound) {
			node := k.CapacityNode[capacity]
			delete(k.NodesIPRange, node)
			delete(k.Keys, node)
			delete(k.WGPort, node)
			delete(k.NodeCapacityMap, node)
			delete(k.CapacityDeploymentMap, capacity)
			delete(k.CapacityNode, capacity)
		} else if err != nil {
			return errors.Wrapf(err, "couldn't get capacity contract id %d ", capacity)
		}
		_, err = sub.GetDeployment(deploymentID)
		if err != nil {
			return fmt.Errorf("couldn't get deployment with id %d. %w", deploymentID, err)
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

	return isNodesUp(ctx, sub, k.NodeIDs, k.ncPool)
}

func (k *NetworkDeployer) ValidateDelete(ctx context.Context) error {
	return nil
}

func (k *NetworkDeployer) storeState(d *schema.ResourceData, state state.StateI) error {

	capacityDeploymentMap := map[string]interface{}{}
	for capacity, deployment := range k.CapacityDeploymentMap {
		capacityDeploymentMap[fmt.Sprint(capacity)] = int(deployment)
	}

	nodesIPRange := make(map[string]interface{})
	for node, r := range k.NodesIPRange {
		nodesIPRange[fmt.Sprint(node)] = r.String()
	}

	nodes := make([]uint32, 0)
	for capacityID := range k.CapacityDeploymentMap {
		nodeID := k.CapacityNode[capacityID]
		if k.PublicNodeID == nodeID {
			continue
		}
		nodes = append(nodes, nodeID)
	}

	nodeCapacity := map[string]interface{}{}
	for nodeID, capacityID := range k.NodeCapacityMap {
		nodeCapacity[fmt.Sprint(nodeID)] = int(capacityID)
	}

	log.Printf("setting deployer object nodes: %v\n", nodes)
	// update network local status
	k.updateNetworkLocalState(state)

	k.NodeIDs = nodes

	log.Printf("storing nodes: %v\n", nodes)
	err := errSet{}
	err.Push(d.Set(NetworkSchemaNodeIDs, nodes))
	err.Push(d.Set(NetworkSchemaIPRange, k.IPRange.String()))
	err.Push(d.Set(NetworkSchemaAccessWGConfig, k.AccessWGConfig))
	if k.ExternalIP == nil {
		err.Push(d.Set(NetworkSchemaExternalIP, nil))
	} else {
		err.Push(d.Set(NetworkSchemaExternalIP, k.ExternalIP.String()))
	}
	err.Push(d.Set(NetworkSchemaExternalSK, k.ExternalSK.String()))
	err.Push(d.Set(NetworkSchemaPublicNodeID, k.PublicNodeID))
	err.Push(d.Set(NetworkSchemaNodesIPRange, nodesIPRange))
	err.Push(d.Set(NetworkSchemaCapacityDeploymentMap, capacityDeploymentMap))
	err.Push(d.Set(NetworkSchemaNodeCapacityMap, nodeCapacity))
	err.Push(d.Set(NetworkSchemaAccessNodeCapacityID, k.AccessNodeCapacityID))
	err.Push(d.Set(NetworkSchemaCapacityIDs, k.CapacityIDs))
	return err.error()
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
	for includes[byte](used, *start) && *start <= 254 {
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
		if includes[uint32](nodes, node) {
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
	capacityDeployments, err := k.deployer.GetDeploymentObjects(ctx, sub, k.CapacityDeploymentMap)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}
	printDeployments(capacityDeployments)
	WGAccess := false
	for capacityID, dl := range capacityDeployments {
		node := k.CapacityNode[capacityID]
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

func (k *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context, sub *substrate.Substrate) (map[uint64]gridtypes.Deployment, error) {
	log.Printf("Node-CapacityID: %v\n", k.NodeCapacityMap)
	deployments := make(map[uint64]gridtypes.Deployment)
	endpoints := make(map[uint32]string)
	hiddenNodes := make([]uint32, 0)
	// var ipv4Node uint32
	accessibleNodes := make([]uint32, 0)
	for node := range k.NodeCapacityMap {
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
			// ipv4Node = node
			endpoints[node] = endpoint.String()
		} else {
			accessibleNodes = append(accessibleNodes, node)
			endpoints[node] = fmt.Sprintf("[%s]", endpoint.String())
		}
	}
	needsIPv4Access := k.AddWGAccess || (len(hiddenNodes) != 0 && len(hiddenNodes)+len(accessibleNodes) > 1)
	if needsIPv4Access {
		// if ipv4 access is needed, k.publicNodeID should always be set to some node
		if !includes[uint32](accessibleNodes, k.PublicNodeID) {
			accessibleNodes = append(accessibleNodes, k.PublicNodeID)
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
		peers := make([]zos.Peer, 0, len(k.NodeCapacityMap))
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
		capacityID := k.NodeCapacityMap[node]
		deployments[capacityID] = deployment
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
		capacityID := k.NodeCapacityMap[node]
		deployments[capacityID] = deployment
	}
	return deployments, nil
}
func (k *NetworkDeployer) Deploy(ctx context.Context, sub *substrate.Substrate) error {
	newDeployments, err := k.GenerateVersionlessDeployments(ctx, sub)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	log.Printf("new deployments")
	printDeployments(newDeployments)
	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
	// currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	if err := k.updateState(ctx, sub, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}
func (k *NetworkDeployer) updateState(ctx context.Context, sub *substrate.Substrate, currentDeploymentIDs map[uint64]uint64) error {
	k.CapacityDeploymentMap = currentDeploymentIDs
	if err := k.readNodesConfig(ctx, sub); err != nil {
		return errors.Wrap(err, "couldn't read node's data")
	}

	return nil
}

func (k *NetworkDeployer) Cancel(ctx context.Context, sub *substrate.Substrate) error {
	newDeployments := make(map[uint64]gridtypes.Deployment)

	currentDeployments, err := k.deployer.Deploy(ctx, sub, k.CapacityDeploymentMap, newDeployments)
	// currentDeployments, err := k.deployer.Deploy(ctx, sub, k.NodeDeploymentID, newDeployments)
	if err := k.updateState(ctx, sub, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := deployer.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}
	if err := deployer.updateAccessNodeCapacity(ctx); err != nil {
		return diag.FromErr(fmt.Errorf("couldn't update access node capacity contract. %w", err))
	}
	err = deployer.Deploy(ctx, apiClient.substrateConn)
	if err != nil {
		if len(deployer.CapacityDeploymentMap) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}
	err = deployer.storeState(d, apiClient.state)
	d.SetId(uuid.New().String())
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("error while storing state", err))...)
	}
	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx, apiClient.substrateConn); err != nil {
		return diag.FromErr(err)
	}
	if err := deployer.updateAccessNodeCapacity(ctx); err != nil {
		return diag.FromErr(fmt.Errorf("couldn't update access node capacity contract. %w", err))
	}
	if err := deployer.invalidateBrokenAttributes(apiClient.substrateConn); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.Deploy(ctx, apiClient.substrateConn)
	if err != nil {
		diags = diag.FromErr(err)
	}
	err = deployer.storeState(d, apiClient.state)
	if err != nil {
		diags = append(diags, diag.FromErr(fmt.Errorf("error while storing state", err))...)
	}
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
	err = deployer.storeState(d, apiClient.state)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error while storing state", err))
	}
	return diags
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	err = deployer.Cancel(ctx, apiClient.substrateConn)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
		ns := apiClient.state.GetNetworkState()
		ns.DeleteNetwork(deployer.Name)
	} else {
		err = deployer.storeState(d, apiClient.state)
		if err != nil {
			diags = append(diags, diag.FromErr(fmt.Errorf("error while storing state", err))...)
		}
	}
	return diags
}

// CreateAccessNodeCapacity is used when a capacity contract is needed for wg access.
//
// an already used farm is preferred to create the capacity contract on. if no farm is eligible, a random farm is selected.
func (k *NetworkDeployer) CreateAccessNodeCapacity() (contractID uint64, err error) {
	// preferebly choose a farm used with the capacity contracts, if not, choose a random one
	farmID, err := k.getPreferrableFarm()
	if err != nil {
		return 0, fmt.Errorf("couldn't get a farm to create a capacity contract on for wg access. %w", err)
	}
	resources := substrate.Resources{
		HRU: 0,
		SRU: 0,
		CRU: 0,
		MRU: 0,
	}
	// log chosen farm
	log.Printf("farm %d was chosen to create capacity contract on for wg access. %d", farmID)

	policy := substrate.WithCapacityPolicy(resources, substrate.NodeFeatures{IsPublicNode: true})
	contractID, err = k.APIClient.substrateConn.CreateCapacityReservationContract(k.APIClient.identity, uint32(farmID), policy, nil)
	if err != nil {
		return 0, fmt.Errorf("couldn't create capacity contract for wg access. %w", err)
	}
	log.Printf("capacity contract id %d was created for wg access", contractID)
	return contractID, nil
}

// getPreferrableFarm gets a farm to create a capacity contract on.
// this farm is preferrably one of the farms already used in provided capacity contracts.
// if non is eligible, a random eligible farm is chosen
func (k *NetworkDeployer) getPreferrableFarm() (uint64, error) {
	freeIP := uint64(1)
	dedicated := false
	for _, id := range k.CapacityIDs {
		contract, err := k.APIClient.substrateConn.GetContract(id)
		if err != nil {
			return 0, fmt.Errorf("couldn't get capacity contract with id %d. %w", id, err)
		}
		nodeID := uint32(contract.ContractType.CapacityReservationContract.NodeID)
		node, err := k.APIClient.substrateConn.GetNode(nodeID)
		if err != nil {
			return 0, fmt.Errorf("couldn't get node with id %d. %w", id, err)
		}
		farmID := uint64(node.FarmID)
		_, cnt, err := k.APIClient.grid_client.Farms(types.FarmFilter{
			FarmID:  &farmID,
			FreeIPs: &freeIP,
		}, types.Limit{
			Size: 1,
		})
		if err != nil {
			return 0, fmt.Errorf("couldn't get farm with id %d. %w", &farmID, err)
		}
		if cnt == 0 {
			log.Printf("farm %d has no free public ips", &farmID)
			continue
		}

		return farmID, nil
	}
	// non is eligible, choose a random eligible farm
	farms, cnt, err := k.APIClient.grid_client.Farms(types.FarmFilter{
		FreeIPs:   &freeIP,
		Dedicated: &dedicated,
	},
		types.Limit{
			Size: 1,
		})
	if cnt == 0 {
		return 0, fmt.Errorf("couldn't get farm with public ips")
	}
	if err != nil {
		return 0, fmt.Errorf("couldn't get farm with public ips. %w", err)
	}
	return uint64(farms[0].FarmID), nil
}

// deleteAccessNodeCapacity deletes the access node capacity contract
func (k *NetworkDeployer) deleteAccessNodeCapacity() error {
	if k.AccessNodeCapacityID == 0 {
		return nil
	}
	err := k.APIClient.substrateConn.CancelContract(k.APIClient.identity, k.AccessNodeCapacityID)
	if err != nil {
		if errors.Is(err, substrate.ErrNotFound) {
			return nil
		} else {
			return fmt.Errorf("couldn't delete access node capacity contract with id %d. %w", k.AccessNodeCapacityID, err)
		}
	}
	return nil
}

// updateAccessNodeCapacity should decide whether to create a new capacity for an access node, or to delete it if not needed any more.
func (k *NetworkDeployer) updateAccessNodeCapacity(ctx context.Context) error {
	// if capacity contract is needed, create it if not created
	atLeastHiddenNode, err := k.hasHiddenNodes(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get network hidden nodes. %w", err)
	}
	if k.AddWGAccess || atLeastHiddenNode {
		// access node is needed
		if k.AccessNodeCapacityID != 0 {
			return nil
		}
		contractID, err := k.CreateAccessNodeCapacity()
		if err != nil {
			return fmt.Errorf("couldn't create a capacity contract for access node. %w", err)
		}

		k.AccessNodeCapacityID = contractID
		contract, err := k.APIClient.substrateConn.GetContract(contractID)
		if err != nil {
			return fmt.Errorf("couldn't retreive capacity contract %d. %w", contractID, err)
		}
		nodeID := uint32(contract.ContractType.CapacityReservationContract.NodeID)

		k.AccessNodeCapacityID = contractID
		k.NodeCapacityMap[nodeID] = contractID
		k.CapacityNode[contractID] = nodeID
	} else {
		if k.AccessNodeCapacityID == 0 {
			return nil
		}
		err := k.deleteAccessNodeCapacity()
		if err != nil {
			return fmt.Errorf("couldn't delete access node's capacity contract with id %d. %w", k.AccessNodeCapacityID, err)
		}
		contractID := k.AccessNodeCapacityID
		nodeID := k.CapacityNode[contractID]
		delete(k.CapacityNode, contractID)
		delete(k.NodeCapacityMap, nodeID)
		k.AccessNodeCapacityID = 0
	}
	return nil
}

// hasHiddenNodes returns a boolean indicating whether or not the set of nodes the network should be deployed on have at least one hidden node.
func (k *NetworkDeployer) hasHiddenNodes(ctx context.Context) (bool, error) {
	for _, node := range k.NodeIDs {
		nodeClient, err := k.ncPool.GetNodeClient(k.APIClient.substrateConn, node)
		if err != nil {
			return false, fmt.Errorf("couldn't get node client. %w", err)
		}
		_, err = getNodeEndpoint(ctx, nodeClient)
		if err != nil && errors.Is(err, ErrNoAccessibleInterfaceFound) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("couldn't get node %d endpoint. %w", node, err)
		}
	}
	return false, nil
}
