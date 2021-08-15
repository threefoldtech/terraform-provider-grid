package provider

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/substrate"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	Substrate = "wss://explorer.devnet.grid.tf/ws"

// Version = 0
// Twin      = 14
// NodeID = 2
// Seed      = "d161de46d136d96085906b9f3d40d08b3649c80a3e4d77f0b14d3dc6889e9dcb"
// Substrate = "wss://explorer.devnet.grid.tf/ws"
// rmb_url   = "tcp://127.0.0.1:6379"
)

func getNodeEndpoint(nodeId int, wgPort int) string {
	nodeIps := make(map[int]string)
	nodeIps[1] = "2a02:1802:5e:16:ec4:7aff:fe30:65b4"
	nodeIps[2] = "2a02:1802:5e:16:225:90ff:fef5:582e"
	nodeIps[3] = "2a02:1802:5e:16:225:90ff:fef5:6d02"
	nodeIps[4] = "2a02:1802:5e:16:225:90ff:fe87:d7b4"

	return fmt.Sprintf("[%s]:%d", nodeIps[nodeId], wgPort)
}

func isIn(l []uint16, i uint16) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInByte(l []byte, i byte) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInInt(l []int, i int) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInStr(l []string, i string) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func getNodClient(nodeId uint32) (*client.NodeClient, error) {
	Substrate := "wss://explorer.devnet.grid.tf/ws"
	cl, err := rmb.NewClient("tcp://127.0.0.1:6379")
	if err != nil {
		return nil, err
	}
	sub, err := substrate.NewSubstrate(Substrate)
	if err != nil {
		return nil, err
	}
	log.Printf("fre node port, node id: %d\n", nodeId)
	nodeInfo, err := sub.GetNode(nodeId)
	if err != nil {
		return nil, err
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)
	return node, nil
}

func getNodeFreeWGPort(ctx context.Context, nodeClient *client.NodeClient, nodeId uint32) (int, error) {
	freeports, err := nodeClient.NetworkListWGPorts(ctx)
	if err != nil {
		return 0, err
	}
	log.Printf("reserved ports for node %d: %v\n", nodeId, freeports)
	p := uint(rand.Intn(6000) + 2000)

	for isIn(freeports, uint16(p)) {
		p = uint(rand.Intn(6000) + 2000)
	}
	log.Printf("Selected port for node %d is %d\n", nodeId, p)
	return int(p), nil
}

func getNetworkFreeIp(fm string) string {
	return fmt.Sprintf(fm, rand.Int31()%254+2)
}

func getPublicNode() int {
	return 1
}

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
			"version": {
				Description: "Version",
				Type:        schema.TypeInt,
				Optional:    true,
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
			"deployment_info": &schema.Schema{
				Type:     schema.TypeList,
				Required: false,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"version": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"deployment_id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"wg_private_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"wg_public_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"wg_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"ip_range": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func ipNet(a, b, c, d, msk byte) gridtypes.IPNet {
	return gridtypes.NewIPNet(net.IPNet{
		IP:   net.IPv4(a, b, c, d),
		Mask: net.CIDRMask(int(msk), 32),
	})
}
func wgIP(ip gridtypes.IPNet) gridtypes.IPNet {
	a := ip.IP[len(ip.IP)-3]
	b := ip.IP[len(ip.IP)-2]

	return gridtypes.NewIPNet(net.IPNet{
		IP:   net.IPv4(100, 64, a, b),
		Mask: net.CIDRMask(32, 32),
	})

}

type NetworkConfiguration struct {
	IPRange         string
	Description     string
	NodeIDs         []int
	Keys            map[int]wgtypes.Key
	Versions        map[int]int
	DeplotmentIDs   map[int]int
	IPs             map[int]gridtypes.IPNet
	WGPort          map[int]int
	PublicNodeID    int
	ExternalNodeIP  gridtypes.IPNet
	ExternalNodeKey wgtypes.Key
}

type DeploymentInformation struct {
	WGAccessConfiguration string
	Deployments           []gridtypes.Deployment
}

type UserIdentityInfo struct {
	TwinID   uint32
	Identity substrate.Identity
	Cl       rmb.Client
	UserSK   ed25519.PrivateKey
}

type NodeDeploymentsInfo struct {
	Version      int
	DeploymentID int
	NodeID       int
	WGPrivateKey string
	WGPublicKey  string
	WGPort       int
	IPRange      string
}

func (ndi *NodeDeploymentsInfo) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["version"] = ndi.Version
	res["node_id"] = ndi.NodeID
	res["deployment_id"] = ndi.DeploymentID
	res["wg_private_key"] = ndi.WGPrivateKey
	res["wg_public_key"] = ndi.WGPublicKey
	res["wg_port"] = ndi.WGPort
	res["ip_range"] = ndi.IPRange
	return res

}

func generateWGConfig(Address string, AccessPrivatekey string, NodePublicKey string, NodeEndpoint string, NetworkIPRange string) string {

	return fmt.Sprintf(`
[Interface]
Address = %s
PrivateKey = %s
[Peer]
PublicKey = %s
AllowedIPs = %s, 100.64.0.0/16
PersistentKeepalive = 25
Endpoint = %s
	`, Address, AccessPrivatekey, NodePublicKey, NetworkIPRange, NodeEndpoint)
}

func (nc *NetworkConfiguration) generateDeployments(ctx context.Context, userInfo *UserIdentityInfo, networkName string) (*DeploymentInformation, error) {

	deploymentInfotmation := &DeploymentInformation{}
	deployments := make([]gridtypes.Deployment, 0)
	for _, node := range nc.NodeIDs {
		peers := make([]zos.Peer, 0, len(nc.NodeIDs))
		for _, neigh := range nc.NodeIDs {
			if node == neigh {
				continue
			}
			neigh_ip_range, _ := nc.IPs[neigh]
			neigh_port, _ := nc.WGPort[neigh]
			neigh_pubkey := nc.Keys[neigh].PublicKey().String()
			allowed_ips := []gridtypes.IPNet{
				neigh_ip_range,
				wgIP(neigh_ip_range),
			}
			if neigh == nc.PublicNodeID {
				allowed_ips = append(allowed_ips, nc.ExternalNodeIP)
				allowed_ips = append(allowed_ips, wgIP(nc.ExternalNodeIP))
			}
			log.Printf("%v\n", allowed_ips)
			peers = append(peers, zos.Peer{
				Subnet:      neigh_ip_range,
				WGPublicKey: neigh_pubkey,
				Endpoint:    getNodeEndpoint(neigh, neigh_port),
				AllowedIPs:  allowed_ips,
			})
		}

		if node == nc.PublicNodeID {
			nodeClient, err := getNodClient(uint32(node))
			if err != nil {
				return nil, err
			}
			publicConfig, err := nodeClient.NetworkGetPublicConfig(ctx)
			if err != nil {
				return nil, err
			}
			l := len(publicConfig.IPv4.IP)
			publicIPStr := fmt.Sprintf("%d.%d.%d.%d", publicConfig.IPv4.IP[l-4], publicConfig.IPv4.IP[l-3], publicConfig.IPv4.IP[l-2], publicConfig.IPv4.IP[l-1])
			externalNodeIPStr := fmt.Sprintf("100.64.%d.%d", publicConfig.IPv4.IP[l-3], publicConfig.IPv4.IP[l-2])
			nodePubky := nc.Keys[node].PublicKey().String()
			WGConfig := generateWGConfig(externalNodeIPStr, nc.ExternalNodeKey.String(), nodePubky, fmt.Sprintf("%s:%d", publicIPStr, nc.WGPort[nc.PublicNodeID]), nc.IPRange)
			log.Printf(WGConfig)
			deploymentInfotmation.WGAccessConfiguration = WGConfig
			peers = append(peers, zos.Peer{
				Subnet:      nc.ExternalNodeIP,
				WGPublicKey: nc.ExternalNodeKey.PublicKey().String(),
				AllowedIPs:  []gridtypes.IPNet{nc.ExternalNodeIP, wgIP(nc.ExternalNodeIP)},
			})
		}
		node_ip_range, ok := nc.IPs[node]
		if !ok {
			return nil, errors.New("couldn't find node ip range in a pre-computed dict of ips")
		}
		node_port, ok := nc.WGPort[node]
		if !ok {
			return nil, errors.New("couldn't find node port in a pre-computed dict of wg ports")
		}
		workload := gridtypes.Workload{
			Version:     0,
			Type:        zos.NetworkType,
			Description: nc.Description,
			Name:        gridtypes.Name(networkName),
			Data: gridtypes.MustMarshal(zos.Network{
				NetworkIPRange: gridtypes.MustParseIPNet(nc.IPRange),
				Subnet:         node_ip_range,
				WGPrivateKey:   nc.Keys[node].String(),
				WGListenPort:   uint16(node_port),
				Peers:          peers,
			}),
		}
		deployment := gridtypes.Deployment{
			Version: Version,
			TwinID:  userInfo.TwinID, //LocalTwin,
			// this contract id must match the one on substrate
			Workloads: []gridtypes.Workload{
				workload,
			},
			SignatureRequirement: gridtypes.SignatureRequirement{
				WeightRequired: 1,
				Requests: []gridtypes.SignatureRequest{
					{
						TwinID: userInfo.TwinID,
						Weight: 1,
					},
				},
			},
		}
		if err := deployment.Valid(); err != nil {
			return nil, err
		}

		if err := deployment.Sign(userInfo.TwinID, userInfo.UserSK); err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	deploymentInfotmation.Deployments = deployments
	return deploymentInfotmation, nil
}
func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	networkName := d.Get("name").(string)
	networkIPRange := d.Get("ip_range").(string)
	networkIP, err := gridtypes.ParseIPNet(networkIPRange)

	if err != nil {
		return diag.FromErr(err)
	}

	identity, err := substrate.IdentityFromPhrase(apiClient.mnemonics)

	if err != nil {
		return diag.FromErr(err)
	}
	userSK, err := identity.SecureKey()
	cl := apiClient.client

	if err != nil {
		return diag.FromErr(err)
	}

	userInfo := &UserIdentityInfo{
		TwinID:   apiClient.twin_id,
		Identity: identity,
		Cl:       cl,
		UserSK:   userSK,
	}

	privateKeys := make(map[int]wgtypes.Key)
	freePort := make(map[int]int)
	ip := make(map[int]gridtypes.IPNet)
	node_ids_ifs := d.Get("nodes").([]interface{})
	public_node := getPublicNode()
	node_ids := make([]int, len(node_ids_ifs))

	for i := range node_ids_ifs {
		node_ids[i] = node_ids_ifs[i].(int)
	}

	external_node_key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return diag.FromErr(err)
	}
	l := len(networkIP.IP)
	external_node_ip := ipNet(networkIP.IP[l-4], networkIP.IP[l-3], 2, 0, 24)

	if err != nil {
		return diag.FromErr(err)
	}
	networkConfig := &NetworkConfiguration{
		Description:     d.Get("description").(string),
		IPRange:         networkIPRange,
		PublicNodeID:    public_node,
		ExternalNodeIP:  external_node_ip,
		ExternalNodeKey: external_node_key,
	}

	stateInfo := make([]NodeDeploymentsInfo, len(node_ids))
	for idx, node := range node_ids {
		nodeClient, err := getNodClient(uint32(node))
		if err != nil {
			return diag.FromErr(err)
		}
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return diag.FromErr(err)
		}
		privateKeys[node] = key
		ip[node] = ipNet(networkIP.IP[l-4], networkIP.IP[l-3], byte(idx+3), 0, 24)
		port, err := getNodeFreeWGPort(ctx, nodeClient, uint32(node))
		if err != nil {
			return diag.FromErr(err)
		}
		freePort[node] = port
		log.Printf("node pubkey: %s, node privkey: %s, node id: %d", key.String(), key.PublicKey(), node)
		stateInfo[idx].Version = 0
		stateInfo[idx].NodeID = node
		stateInfo[idx].IPRange = ip[node].String()
		stateInfo[idx].WGPort = freePort[node]
		stateInfo[idx].WGPrivateKey = privateKeys[node].String()
		stateInfo[idx].WGPublicKey = privateKeys[node].PublicKey().String()
	}
	networkConfig.IPs = ip
	networkConfig.WGPort = freePort
	networkConfig.Keys = privateKeys
	networkConfig.NodeIDs = node_ids
	deployments, err := networkConfig.generateDeployments(ctx, userInfo, networkName)
	if err != nil {
		return diag.FromErr(err)
	}
	for idx, deployment := range deployments.Deployments {
		node := node_ids[idx]
		hash, err := deployment.ChallengeHash()
		if err != nil {
			return diag.FromErr(err)
		}

		hashHex := hex.EncodeToString(hash)
		sub, err := substrate.NewSubstrate(Substrate)
		if err != nil {
			return diag.FromErr(err)
		}

		nodeClient, err := getNodClient(uint32(node))

		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
		defer cancel()
		log.Printf("creating conract, node: %d, hash: %s\n", node, hashHex)
		contractID, err := sub.CreateContract(&identity, uint32(node), nil, hashHex, 0)
		if err != nil {
			return diag.FromErr(err)
		}
		deployment.ContractID = contractID // from substrate
		err = nodeClient.DeploymentDeploy(ctx, deployment)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("node: %d, contract: %d", node, contractID)

		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(deployment)
		stateInfo[idx].DeploymentID = int(contractID)
	}
	StoreState(d, stateInfo)
	d.Set("public_node_id", public_node)
	d.Set("access_wg_config", deployments.WGAccessConfiguration)
	d.Set("external_ip", external_node_ip.String())
	d.Set("external_sk", external_node_key.String())
	d.SetId(uuid.New().String())
	return nil
}

func StoreState(d *schema.ResourceData, stateInfo []NodeDeploymentsInfo) {
	encoded := make([]map[string]interface{}, 0)
	for _, info := range stateInfo {
		encoded = append(encoded, info.Dictify())
	}
	d.Set("deployment_info", encoded)
}

func loadState(d *schema.ResourceData) []NodeDeploymentsInfo {
	encoded := d.Get("deployment_info").([]interface{})
	stateInfo := make([]NodeDeploymentsInfo, 0)
	for _, infoI := range encoded {
		info := infoI.(map[string]interface{})
		stateInfo = append(stateInfo, NodeDeploymentsInfo{
			Version:      info["version"].(int),
			DeploymentID: info["deployment_id"].(int),
			NodeID:       info["node_id"].(int),
			WGPrivateKey: info["wg_private_key"].(string),
			WGPublicKey:  info["wg_public_key"].(string),
			WGPort:       info["wg_port"].(int),
			IPRange:      info["ip_range"].(string),
		})
	}
	return stateInfo
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

func loadNetworkConfig(d *schema.ResourceData, stateInfo []NodeDeploymentsInfo) (*NetworkConfiguration, error) {
	log.Printf("Fetched node key: %s\n", d.Get("external_sk").(string))
	nodeKey, err := wgtypes.ParseKey(d.Get("external_sk").(string))
	if err != nil {
		return nil, err
	}
	networkConfig := &NetworkConfiguration{
		Description:     d.Get("description").(string),
		IPRange:         d.Get("ip_range").(string),
		NodeIDs:         make([]int, 0),
		Keys:            make(map[int]wgtypes.Key),
		IPs:             make(map[int]gridtypes.IPNet),
		WGPort:          make(map[int]int),
		Versions:        make(map[int]int),
		DeplotmentIDs:   make(map[int]int),
		PublicNodeID:    d.Get("public_node_id").(int),
		ExternalNodeIP:  gridtypes.MustParseIPNet(d.Get("external_ip").(string)),
		ExternalNodeKey: nodeKey,
	}
	for _, info := range stateInfo {
		networkConfig.NodeIDs = append(networkConfig.NodeIDs, info.NodeID)
		networkConfig.Keys[info.NodeID], err = wgtypes.ParseKey(info.WGPrivateKey)
		if err != nil {
			return nil, err
		}
		networkConfig.IPs[info.NodeID] = gridtypes.MustParseIPNet(info.IPRange)
		networkConfig.WGPort[info.NodeID] = info.WGPort
		networkConfig.Versions[info.NodeID] = info.Version
		networkConfig.DeplotmentIDs[info.NodeID] = info.DeploymentID
	}
	return networkConfig, nil
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	networkName := d.Get("name").(string)
	identity, err := substrate.IdentityFromPhrase(apiClient.mnemonics)

	if err != nil {
		return diag.FromErr(err)
	}
	userSK, err := identity.SecureKey()
	cl := apiClient.client

	if err != nil {
		return diag.FromErr(err)
	}

	userInfo := &UserIdentityInfo{
		TwinID:   apiClient.twin_id,
		Identity: identity,
		Cl:       cl,
		UserSK:   userSK,
	}

	stateInfo := loadState(d)
	node_ids_ifs := d.Get("nodes").([]interface{})
	node_ids := make([]int, len(node_ids_ifs))
	networkConfig, err := loadNetworkConfig(d, stateInfo)
	if err != nil {
		return diag.FromErr(err)
	}
	network_ip := gridtypes.MustParseIPNet(d.Get("ip_range").(string))
	l := len(network_ip.IP)
	log.Printf("network ip range: %v\n", network_ip)
	usedIps := make([]byte, 0) // the third octet
	usedIps = append(usedIps, networkConfig.ExternalNodeIP.IP[l-2])
	for _, ip := range networkConfig.IPs {
		usedIps = append(usedIps, ip.IP[l-2])
	}
	for i := range node_ids_ifs {
		node_ids[i] = node_ids_ifs[i].(int)
	}

	var cur byte = 3
	for _, node := range node_ids {
		if _, exists := networkConfig.WGPort[node]; !exists {
			for isInByte(usedIps, cur) {
				cur += 1
			}
			networkConfig.NodeIDs = append(networkConfig.NodeIDs, node)
			networkConfig.IPs[node] = ipNet(network_ip.IP[l-4], network_ip.IP[l-3], cur, network_ip.IP[l-2], 24)
			// log.Printf("ip is: %d %d %d %d\n", network_ip.IP[l - 4], network_ip.IP[l - 3], cur, network_ip.IP[l - 1])
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return diag.FromErr(err)
			}
			networkConfig.Keys[node] = key
			nodeClient, err := getNodClient(uint32(node))
			if err != nil {
				return diag.FromErr(err)
			}
			port, err := getNodeFreeWGPort(ctx, nodeClient, uint32(node))
			if err != nil {
				return diag.FromErr(err)
			}
			networkConfig.WGPort[node] = port
			networkConfig.Versions[node] = -1
			networkConfig.DeplotmentIDs[node] = -1
			cur += 1
		}
	}

	newStateInfo := make([]NodeDeploymentsInfo, len(node_ids))
	for idx, node := range node_ids {

		newStateInfo[idx].Version = 0
		if ver, ok := networkConfig.Versions[node]; ok {
			newStateInfo[idx].Version = ver
		}
		newStateInfo[idx].NodeID = node
		newStateInfo[idx].IPRange = networkConfig.IPs[node].String()
		newStateInfo[idx].WGPort = networkConfig.WGPort[node]
		newStateInfo[idx].WGPrivateKey = networkConfig.Keys[node].String()
		newStateInfo[idx].WGPublicKey = networkConfig.Keys[node].PublicKey().String()
		newStateInfo[idx].WGPublicKey = networkConfig.Keys[node].PublicKey().String()
	}
	networkConfig.NodeIDs = node_ids
	deployments, err := networkConfig.generateDeployments(ctx, userInfo, networkName)
	if err != nil {
		return diag.FromErr(err)
	}
	for idx, deployment := range deployments.Deployments {
		sub, err := substrate.NewSubstrate(Substrate)
		if err != nil {
			return diag.FromErr(err)
		}
		node := node_ids[idx]

		if err != nil {
			return diag.FromErr(err)
		}

		ver, _ := networkConfig.Versions[node]
		deployment.Version = ver + 1
		deployment.Workloads[0].Version = ver + 1
		newStateInfo[idx].Version = ver + 1
		if err := deployment.Valid(); err != nil {
			return diag.FromErr(err)
		}

		if err := deployment.Sign(apiClient.twin_id, userSK); err != nil {
			return diag.FromErr(err)
		}

		hash, err := deployment.ChallengeHash()
		if err != nil {
			return diag.FromErr(err)
		}
		hashHex := hex.EncodeToString(hash)

		contractID, err := uint64(0), error(nil)
		if networkConfig.Versions[node] == -1 {
			contractID, err = sub.CreateContract(&identity, uint32(node), nil, hashHex, 0)
			log.Printf("Creating contract %d\n", contractID)
		} else {

			deploymentID, _ := networkConfig.DeplotmentIDs[node]
			log.Printf("Updating contract %d\n", deploymentID)
			contractID, err = sub.UpdateContract(&identity, uint64(deploymentID), nil, hashHex)
		}

		if err != nil {
			return diag.FromErr(err)
		}

		deployment.ContractID = contractID // from substrate

		nodeClient, err := getNodClient(uint32(node))
		if err != nil {
			return diag.FromErr(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
		defer cancel()
		if ver == -1 {
			log.Printf("Creating deployment\n")
			err = nodeClient.DeploymentDeploy(ctx, deployment)
		} else {
			log.Printf("Updating deployment %v\n", deployment)
			err = nodeClient.DeploymentUpdate(ctx, deployment)
		}
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("node: %d, contract: %d", node, contractID)

		got, err := nodeClient.DeploymentGet(ctx, deployment.ContractID)
		if err != nil {
			return diag.FromErr(err)
		}
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(got)
		newStateInfo[idx].DeploymentID = int(contractID)
	}
	for _, info := range stateInfo {
		if !isInInt(node_ids, info.NodeID) {
			node := info.NodeID
			sub, err := substrate.NewSubstrate(Substrate)
			if err != nil {
				return diag.FromErr(err)
			}

			nodeInfo, err := sub.GetNode(uint32(node))
			if err != nil {
				return diag.FromErr(err)
			}

			node_client := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)
			sub.CancelContract(&identity, uint64(info.DeploymentID))
			node_client.DeploymentDelete(ctx, uint64(info.DeploymentID))
			fmt.Printf("deleting %d\n", info.DeploymentID)
		}
	}
	StoreState(d, newStateInfo)

	return diag.Diagnostics{}
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)

	stateInfo := loadState(d)
	identity, err := substrate.IdentityFromPhrase(apiClient.mnemonics)

	if err != nil {
		return diag.FromErr(err)
	}

	cl, err := rmb.NewClient("tcp://127.0.0.1:6379")
	if err != nil {
		return diag.FromErr(err)
	}
	sub, err := substrate.NewSubstrate(Substrate)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, info := range stateInfo {
		cid := uint64(info.DeploymentID)
		nodeIDint := info.NodeID
		if err != nil {
			return diag.FromErr(err)
		}
		nodeInfo, err := sub.GetNode(uint32(nodeIDint))
		if err != nil {
			return diag.FromErr(err)
		}
		node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)
		ctx := context.Background()

		err = sub.CancelContract(&identity, cid)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := node.DeploymentDelete(ctx, cid); err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
}
