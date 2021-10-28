package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const ExternalNodeID = -1

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Network Name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "Description field",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"nodes": {
				Description: "List of nodes to add to the network",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"ip_range": {
				Description: "Network ip range",
				Type:        schema.TypeString,
				Required:    true,
			},
			"add_wg_access": {
				Description: "Whether to add a public node to network and use it to generate a wg config",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"access_wg_config": {
				Description: "WG config for access",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"external_ip": {
				Description: "IP of the access point",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"external_sk": {
				Description: "Access point private key",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"public_node_id": {
				Description: "Public node id (in case it's added)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"nodes_ip_range": {
				Description: "Computed values of nodes' ip ranges after deployment",
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"node_deployment_id": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

type NetworkDeployer struct {
	Name        string
	Description string
	Nodes       []uint32
	IPRange     gridtypes.IPNet
	AddWGAccess bool

	AccessWGConfig   string
	ExternalIP       *gridtypes.IPNet
	ExternalSK       wgtypes.Key
	PublicNodeID     uint32
	NodeDeploymentID map[uint32]uint64
	NodesIPRange     map[uint32]gridtypes.IPNet

	WGPort    map[uint32]int
	Keys      map[uint32]wgtypes.Key
	APIClient *apiClient
	ncPool    *NodeClientPool
}

func NewNetworkDeployer(ctx context.Context, d *schema.ResourceData, apiClient *apiClient) (NetworkDeployer, error) {
	var err error
	nodesIf := d.Get("nodes").([]interface{})
	nodes := make([]uint32, len(nodesIf))
	for idx, n := range nodesIf {
		nodes[idx] = uint32(n.(int))
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
	ncPool := NewNodeClient(apiClient.sub, apiClient.rmb)
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
	deployer := NetworkDeployer{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Nodes:            nodes,
		IPRange:          ipRange,
		AddWGAccess:      addWGAccess,
		AccessWGConfig:   d.Get("access_wg_config").(string),
		ExternalIP:       externalIP,
		ExternalSK:       externalSK,
		PublicNodeID:     uint32(d.Get("public_node_id").(int)),
		NodesIPRange:     nodesIPRange,
		NodeDeploymentID: nodeDeploymentID,
		Keys:             make(map[uint32]wgtypes.Key),
		WGPort:           make(map[uint32]int),
		ncPool:           ncPool,
		APIClient:        apiClient,
	}
	return deployer, nil
}

// invalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *NetworkDeployer) invalidateBrokenAttributes() error {
	for node, contractID := range k.NodeDeploymentID {
		contract, err := k.APIClient.sub.GetContract(contractID)
		if (err != nil && !contract.State.IsCreated) || errors.Is(err, substrate.ErrNotFound) {
			delete(k.NodeDeploymentID, node)
			delete(k.NodesIPRange, node)
			delete(k.Keys, node)
			delete(k.WGPort, node)
		}
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d contract %d", node, contractID)
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
		cl, err := k.ncPool.getNodeClient(k.PublicNodeID)
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
func (k *NetworkDeployer) Validate(ctx context.Context) error {
	if err := validateAccountMoneyForExtrinsics(k.APIClient); err != nil {
		return err
	}
	return isNodesUp(ctx, k.Nodes, k.ncPool)
}

func (k *NetworkDeployer) ValidateDelete(ctx context.Context) error {
	return nil
}

func (k *NetworkDeployer) storeState(d *schema.ResourceData) {

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
func (k *NetworkDeployer) assignNodesWGPort(ctx context.Context, nodes []uint32) error {
	for _, node := range nodes {
		if _, ok := k.WGPort[node]; !ok {
			cl, err := k.ncPool.getNodeClient(node)
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
func (k *NetworkDeployer) readNodesConfig(ctx context.Context) error {
	keys := make(map[uint32]wgtypes.Key)
	WGPort := make(map[uint32]int)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	log.Printf("reading node config")
	nodeDeployments, err := getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
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

func (k *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	log.Printf("nodes: %v\n", k.Nodes)
	deployments := make(map[uint32]gridtypes.Deployment)
	endpoints := make(map[uint32]string)
	hiddenNodes := make([]uint32, 0)
	var ipv4Node uint32
	accessibleNodes := make([]uint32, 0)
	for _, node := range k.Nodes {
		cl, err := k.ncPool.getNodeClient(node)
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
	if needsIPv4Access && ipv4Node == 0 { // we need an ipv4, add one forcibly
		publicNode, err := getPublicNode(ctx, k.ncPool, k.APIClient.graphql_url, []uint32{})
		if err != nil {
			return nil, errors.Wrap(err, "public node needed because you requested adding wg access or a hidden node is added to the network")
		}
		ipv4Node = publicNode
		accessibleNodes = append(accessibleNodes, publicNode)
		cl, err := k.ncPool.getNodeClient(publicNode)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't get node %d client", publicNode)
		}
		endpoint, err := getNodeEndpoint(ctx, cl)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get node %d endpoint", publicNode)
		}
		endpoints[publicNode] = endpoint.String()
	}
	if needsIPv4Access {
		k.PublicNodeID = ipv4Node
	}
	all := append(hiddenNodes, accessibleNodes...)
	if err := k.assignNodesIPs(all); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node ips")
	}
	if err := k.assignNodesWGKey(all); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node wg keys")
	}
	if err := k.assignNodesWGPort(ctx, all); err != nil {
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
	log.Printf("public node: %v\n", ipv4Node)
	log.Printf("accessible nodes: %v\n", accessibleNodes)
	log.Printf("non accessible ip ranges: %v\n", nonAccessibleIPRanges)

	if k.AddWGAccess {
		k.AccessWGConfig = generateWGConfig(
			wgIP(*k.ExternalIP).IP.String(),
			k.ExternalSK.String(),
			k.Keys[ipv4Node].PublicKey().String(),
			fmt.Sprintf("%s:%d", endpoints[ipv4Node], k.WGPort[k.PublicNodeID]),
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
			if neigh == ipv4Node {
				allowed_ips = append(allowed_ips, nonAccessibleIPRanges...)
			}
			peers = append(peers, zos.Peer{
				Subnet:      k.NodesIPRange[neigh],
				WGPublicKey: k.Keys[neigh].PublicKey().String(),
				Endpoint:    fmt.Sprintf("%s:%d", endpoints[neigh], k.WGPort[neigh]),
				AllowedIPs:  allowed_ips,
			})
		}
		if node == ipv4Node {
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
			Version: Version,
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
		deployments[node] = deployment
	}
	// hidden nodes deployments
	for _, node := range hiddenNodes {
		nodeIPRange := k.NodesIPRange[node]
		peers := make([]zos.Peer, 0)
		if ipv4Node != 0 {
			peers = append(peers, zos.Peer{
				WGPublicKey: k.Keys[ipv4Node].PublicKey().String(),
				Subnet:      nodeIPRange,
				AllowedIPs: []gridtypes.IPNet{
					k.IPRange,
					ipNet(100, 64, 0, 0, 16),
				},
				Endpoint: fmt.Sprintf("%s:%d", endpoints[ipv4Node], k.WGPort[ipv4Node]),
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
			Version: Version,
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
		deployments[node] = deployment
	}
	return deployments, nil
}
func (k *NetworkDeployer) Deploy(ctx context.Context) error {
	newDeployments, err := k.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't generate deployments data")
	}
	log.Printf("new deployments")
	printDeployments(newDeployments)
	currentDeployments, err := deployDeployments(ctx, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, true)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}
func (k *NetworkDeployer) updateState(ctx context.Context, currentDeploymentIDs map[uint32]uint64) error {
	k.NodeDeploymentID = currentDeploymentIDs
	if err := k.readNodesConfig(ctx); err != nil {
		return errors.Wrap(err, "couldn't read node's data")
	}

	return nil
}

func (k *NetworkDeployer) Cancel(ctx context.Context) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)

	currentDeployments, err := deployDeployments(ctx, k.NodeDeploymentID, newDeployments, k.ncPool, k.APIClient, false)
	if err := k.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return err
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := deployer.Validate(ctx); err != nil {
		return diag.FromErr(err)
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

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.Validate(ctx); err != nil {
		return diag.FromErr(err)
	}
	if err := deployer.invalidateBrokenAttributes(); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.Deploy(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.invalidateBrokenAttributes(); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = deployer.readNodesConfig(ctx)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   err.Error(),
		})
		return diags
	}
	deployer.storeState(d)
	return diags
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
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
