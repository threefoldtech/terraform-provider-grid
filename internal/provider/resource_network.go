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
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

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
				Required:    true,
			},
			"nodes": {
				Description: "Network size in Gigabytes",
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
			"access_wg_config": {
				Description: "wg config for access",
				Type:        schema.TypeString,
				Required:    false,
				Computed:    true,
			},
			"external_ip": {
				Description: "ip of the access point",
				Type:        schema.TypeString,
				Required:    false,
				Computed:    true,
			},
			"external_sk": {
				Description: "access point private key",
				Type:        schema.TypeString,
				Required:    false,
				Computed:    true,
			},
			"public_node_id": {
				Description: "access point public key",
				Type:        schema.TypeInt,
				Required:    false,
				Computed:    true,
			},
			"nodes_ip_range": {
				Description: "Computed values of nodes' ip ranges after deployment",
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Required:    false,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"node_deployment_id": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			// "deployment_info": {

			// 	Type:     schema.TypeList,
			// 	Required: false,
			// 	Computed: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"node_id": {
			// 				Type:     schema.TypeInt,
			// 				Required: true,
			// 			},
			// 			"version": {
			// 				Type:     schema.TypeInt,
			// 				Required: true,
			// 			},
			// 			"deployment_id": {
			// 				Type:     schema.TypeInt,
			// 				Required: true,
			// 			},
			// 			"wg_private_key": {
			// 				Type:     schema.TypeString,
			// 				Required: true,
			// 			},
			// 			"wg_public_key": {
			// 				Type:     schema.TypeString,
			// 				Required: true,
			// 			},
			// 			"wg_port": {
			// 				Type:     schema.TypeInt,
			// 				Required: true,
			// 			},
			// 			"ip_range": {
			// 				Type:     schema.TypeString,
			// 				Required: true,
			// 			},
			// 		},
			// 	},
			// },
		},
	}
}

type NetworkDeployer struct {
	Name        string
	Description string
	Nodes       []uint32
	IPRange     gridtypes.IPNet

	AccessWGConfig   string
	ExternalIP       *gridtypes.IPNet
	ExternalSK       wgtypes.Key
	PublicNodeID     uint32
	NodeDeploymentID map[uint32]uint64
	NodesIPRange     map[uint32]gridtypes.IPNet

	WGPort                  map[uint32]int
	Keys                    map[uint32]wgtypes.Key
	PublicNodeForceblyAdded bool
	NodeDeployments         map[uint32]gridtypes.Deployment
	APIClient               *apiClient
	ncPool                  *NodeClientPool
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
	publicNodeForceblyAdded := false

	publicNodeID := uint32(d.Get("public_node_id").(int))
	if publicNodeID == 0 {
		nd, err := getPublicNode(nodes)
		if err != nil {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't find node id")
		}

		publicNodeID = nd
	}
	if !isInUint32(nodes, publicNodeID) {
		publicNodeForceblyAdded = true
		nodes = append(nodes, publicNodeID)
	}

	var externalIP *gridtypes.IPNet
	externalIPStr := d.Get("external_ip").(string)
	if externalIPStr != "" {
		ip, err := gridtypes.ParseIPNet(externalIPStr)
		externalIP = &ip
		nodesIPRange[publicNodeID] = *externalIP
		if err != nil && externalIPStr != "" {
			return NetworkDeployer{}, errors.Wrap(err, "couldn't parse external ip")
		}
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
	if err != nil && externalIPStr != "" {
		return NetworkDeployer{}, errors.Wrap(err, "couldn't parse network ip range")
	}
	deployer := NetworkDeployer{
		Name:                    d.Get("name").(string),
		Description:             d.Get("description").(string),
		Nodes:                   nodes,
		IPRange:                 ipRange,
		AccessWGConfig:          d.Get("access_wg_config").(string),
		ExternalIP:              externalIP,
		ExternalSK:              externalSK,
		PublicNodeID:            publicNodeID,
		NodesIPRange:            nodesIPRange,
		NodeDeploymentID:        nodeDeploymentID,
		Keys:                    make(map[uint32]wgtypes.Key),
		WGPort:                  make(map[uint32]int),
		ncPool:                  NewNodeClient(apiClient.sub, apiClient.rmb),
		APIClient:               apiClient,
		PublicNodeForceblyAdded: publicNodeForceblyAdded,
	}
	return deployer, nil
}

func (k *NetworkDeployer) fetchDeploymentsInfo(ctx context.Context) error {
	if len(k.NodeDeploymentID) != 0 {
		deployments, err := getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
		if err != nil {
			return errors.Wrap(err, "couldn't fetch deployments data")
		}
		k.NodeDeployments = deployments

		if err := k.readNodesConfig(); err != nil {
			return errors.Wrap(err, "couldn't read nodes config")
		}
	}
	return nil
}

func (k *NetworkDeployer) ValidateCreate(ctx context.Context) error {
	return isNodesUp(ctx, k.Nodes, k.ncPool)
}

func (k *NetworkDeployer) ValidateUpdate(ctx context.Context) error {
	nodes := make([]uint32, 0)
	nodes = append(nodes, k.Nodes...)
	for node, _ := range k.NodeDeploymentID {
		nodes = append(nodes, node)
	}
	return isNodesUp(ctx, nodes, k.ncPool)
}

func (k *NetworkDeployer) ValidateRead(ctx context.Context) error {
	nodes := make([]uint32, 0)
	for node, _ := range k.NodeDeploymentID {
		nodes = append(nodes, node)
	}
	return isNodesUp(ctx, nodes, k.ncPool)
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
		if _, ok := k.NodeDeployments[node]; ok {
			if k.PublicNodeID == node && k.PublicNodeForceblyAdded {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	for node := range k.NodeDeployments {
		if !isInUint32(nodes, node) {
			if k.PublicNodeID == node && k.PublicNodeForceblyAdded {
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
	d.Set("external_ip", k.ExternalIP.String())
	d.Set("external_sk", k.ExternalSK.String())
	d.Set("public_node_id", k.PublicNodeID)
	// plural or singular?
	d.Set("nodes_ip_range", nodesIPRange)
	d.Set("node_deployment_id", nodeDeploymentID)
}

func (k *NetworkDeployer) assignNodesIPs() error {
	l := len(k.IPRange.IP)
	usedIPs := make([]byte, 0) // the third octet
	for _, ip := range k.NodesIPRange {
		usedIPs = append(usedIPs, ip.IP[l-2])
	}
	var cur byte = 2
	if k.ExternalIP != nil {
		usedIPs = append(usedIPs, k.ExternalIP.IP[l-2])
	} else {
		for isInByte(usedIPs, cur) && cur <= 254 {
			cur += 1
		}
		if cur > 254 {
			return errors.New("couldn't find a free ip to add node")
		}
		ip := ipNet(k.IPRange.IP[l-4], k.IPRange.IP[l-3], cur, k.IPRange.IP[l-2], 24)
		k.ExternalIP = &ip
		usedIPs = append(usedIPs, cur)
		cur += 1
	}

	for _, node := range k.Nodes {
		if _, ok := k.NodesIPRange[node]; !ok {
			for isInByte(usedIPs, cur) && cur <= 254 {
				cur += 1
			}
			if cur > 254 {
				return errors.New("couldn't find a free ip to add node")
			}
			k.NodesIPRange[node] = ipNet(k.IPRange.IP[l-4], k.IPRange.IP[l-3], cur, k.IPRange.IP[l-2], 24)
			usedIPs = append(usedIPs, cur)
			cur += 1
		}
	}
	return nil
}
func (k *NetworkDeployer) assignNodesWGPort(ctx context.Context) error {
	for _, node := range k.Nodes {
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
func (k *NetworkDeployer) assignNodesWGKey() error {
	for _, node := range k.Nodes {
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
func (k *NetworkDeployer) readNodesConfig() error {
	keys := make(map[uint32]wgtypes.Key)
	WGPort := make(map[uint32]int)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	log.Printf("reading node config")
	printDeployments(k.NodeDeployments)

	for node, dl := range k.NodeDeployments {
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
		}
	}
	k.Keys = keys
	k.WGPort = WGPort
	k.NodesIPRange = nodesIPRange
	return nil
}

func (k *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	log.Printf("nodes: %v\n", k.Nodes)
	if err := k.assignNodesIPs(); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node ips")
	}
	if err := k.assignNodesWGKey(); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node wg keys")
	}
	if err := k.assignNodesWGPort(ctx); err != nil {
		return nil, errors.Wrap(err, "couldn't assign node wg ports")
	}
	deployments := make(map[uint32]gridtypes.Deployment)
	for _, node := range k.Nodes {
		nodeClient, err := k.ncPool.getNodeClient(uint32(node))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get node client")
		}
		peers := make([]zos.Peer, 0, len(k.Nodes))
		for _, neigh := range k.Nodes {
			if node == neigh {
				continue
			}
			neigh_ip_range := k.NodesIPRange[neigh]
			neigh_port := k.WGPort[neigh]
			neigh_pubkey := k.Keys[neigh].PublicKey().String()
			neighClient, err := k.ncPool.getNodeClient(uint32(neigh))
			if err != nil {
				return nil, errors.Wrap(err, "coudn't get neighnbor node client")
			}
			allowed_ips := []gridtypes.IPNet{
				neigh_ip_range,
				wgIP(neigh_ip_range),
			}
			if neigh == k.PublicNodeID {
				allowed_ips = append(allowed_ips, *k.ExternalIP)
				allowed_ips = append(allowed_ips, wgIP(*k.ExternalIP))
			}
			log.Printf("%v\n", allowed_ips)
			endpoint, err := getNodeEndpoint(ctx, neighClient)
			if err != nil {
				return nil, errors.Wrap(err, "couldn't get node endpoint")
			}
			peers = append(peers, zos.Peer{
				Subnet:      neigh_ip_range,
				WGPublicKey: neigh_pubkey,
				Endpoint:    fmt.Sprintf("%s:%d", endpoint, neigh_port),
				AllowedIPs:  allowed_ips,
			})
		}

		if node == k.PublicNodeID {
			publicConfig, err := nodeClient.NetworkGetPublicConfig(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public config")
			}
			l := len(publicConfig.IPv4.IP)
			ip := wgIP(*k.ExternalIP)
			publicIPStr := fmt.Sprintf("%d.%d.%d.%d", publicConfig.IPv4.IP[l-4], publicConfig.IPv4.IP[l-3], publicConfig.IPv4.IP[l-2], publicConfig.IPv4.IP[l-1])
			externalNodeIPStr := fmt.Sprintf("100.64.%d.%d", ip.IP[l-2], ip.IP[l-1])
			nodePubky := k.Keys[node].PublicKey().String()
			WGConfig := generateWGConfig(externalNodeIPStr, k.ExternalSK.String(), nodePubky, fmt.Sprintf("%s:%d", publicIPStr, k.WGPort[k.PublicNodeID]), k.IPRange.String())
			log.Printf("%s\n", WGConfig)
			k.AccessWGConfig = WGConfig
			peers = append(peers, zos.Peer{
				Subnet:      *k.ExternalIP,
				WGPublicKey: k.ExternalSK.PublicKey().String(),
				AllowedIPs:  []gridtypes.IPNet{*k.ExternalIP, wgIP(*k.ExternalIP)},
			})
		}
		node_ip_range, ok := k.NodesIPRange[node]
		if !ok {
			return nil, errors.New("couldn't find node ip range in a pre-computed dict of ips")
		}
		node_port, ok := k.WGPort[node]
		if !ok {
			return nil, errors.New("couldn't find node port in a pre-computed dict of wg ports")
		}
		workload := gridtypes.Workload{
			Version:     0,
			Type:        zos.NetworkType,
			Description: k.Description,
			Name:        gridtypes.Name(k.Name),
			Data: gridtypes.MustMarshal(zos.Network{
				NetworkIPRange: gridtypes.MustParseIPNet(k.IPRange.String()),
				Subnet:         node_ip_range,
				WGPrivateKey:   k.Keys[node].String(),
				WGListenPort:   uint16(node_port),
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

func (k *NetworkDeployer) GetOldDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	return getDeploymentObjects(ctx, k.NodeDeploymentID, k.ncPool)
}

func (k *NetworkDeployer) Deploy(ctx context.Context) error {
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
func (k *NetworkDeployer) updateState(ctx context.Context, currentDeploymentIDs map[uint32]uint64) error {
	k.NodeDeploymentID = currentDeploymentIDs
	dls, err := getDeploymentObjects(ctx, currentDeploymentIDs, k.ncPool)
	k.NodeDeployments = dls
	if err != nil {
		return errors.Wrap(err, "couldn't read deployments data")
	}
	if err := k.readNodesConfig(); err != nil {
		return errors.Wrap(err, "couldn't read node's data")
	}

	return nil
}

func (k *NetworkDeployer) removeDeletedContracts(ctx context.Context) error {
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

func (k *NetworkDeployer) updateFromRemote(ctx context.Context) error {
	if err := k.removeDeletedContracts(ctx); err != nil {
		return errors.Wrap(err, "failed to remove deleted contracts")
	}
	return k.readNodesConfig()
}

func (k *NetworkDeployer) Cancel(ctx context.Context) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	oldDeployments := make(map[uint32]gridtypes.Deployment)
	for node, deploymentID := range k.NodeDeploymentID {
		oldDeployments[node] = gridtypes.Deployment{
			ContractID: deploymentID,
		}
	}

	currentDeployments, err := deployDeployments(ctx, oldDeployments, newDeployments, k.ncPool, k.APIClient, false)
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
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}
	if err := deployer.ValidateCreate(ctx); err != nil {
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

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.ValidateUpdate(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}
	if err := deployer.fetchDeploymentsInfo(ctx); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't fetch deployments info"))
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
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	deployer, err := NewNetworkDeployer(ctx, d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	if err := deployer.ValidateRead(ctx); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error happened while doing initial check (check https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)",
			Detail:   err.Error(),
		})
		return diags
	}
	if err := deployer.fetchDeploymentsInfo(ctx); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't fetch deployments info"))
	}

	err = deployer.updateFromRemote(ctx)
	log.Printf("read updateFromRemote err: %s\n", err)
	if err != nil {
		return diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
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
