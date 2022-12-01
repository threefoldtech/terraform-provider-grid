package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	client "github.com/threefoldtech/terraform-provider-grid/pkg/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	trueVal  = true
	statusUp = "up"

	ErrNoAccessibleInterfaceFound = fmt.Errorf("couldn't find a publicly accessible ipv4 or ipv6")
)

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

func getPublicNode(ctx context.Context, gridClient proxy.Client, preferedNodes []uint32) (uint32, error) {
	preferedNodesSet := make(map[int]struct{})
	for _, node := range preferedNodes {
		preferedNodesSet[int(node)] = struct{}{}
	}
	nodes, _, err := gridClient.Nodes(proxytypes.NodeFilter{
		IPv4:   &trueVal,
		Status: &statusUp,
	}, proxytypes.Limit{})
	if err != nil {
		return 0, errors.Wrap(err, "couldn't fetch nodes from the rmb proxy")
	}
	// force add preferred nodes
	nodeMap := make(map[int]struct{})
	for _, node := range nodes {
		nodeMap[node.NodeID] = struct{}{}
	}
	for _, node := range preferedNodes {
		if _, ok := nodeMap[int(node)]; ok {
			continue
		}
		nodeInfo, err := gridClient.Node(node)
		if err != nil {
			log.Printf("failed to get node %d from the grid proxy", node)
			continue
		}
		if nodeInfo.PublicConfig.Ipv4 == "" {
			continue
		}
		if nodeInfo.Status != "up" {
			continue
		}
		nodes = append(nodes, proxytypes.Node{
			PublicConfig: nodeInfo.PublicConfig,
		})
	}
	lastPrefered := 0
	for i := range nodes {
		if _, ok := preferedNodesSet[nodes[i].NodeID]; ok {
			nodes[i], nodes[lastPrefered] = nodes[lastPrefered], nodes[i]
			lastPrefered++
		}
	}
	for _, node := range nodes {
		log.Printf("found a node with ipv4 public config: %d %s\n", node.NodeID, node.PublicConfig.Ipv4)
		ip, _, err := net.ParseCIDR(node.PublicConfig.Ipv4)
		if err != nil {
			log.Printf("couldn't parse public ip %s of node %d: %s", node.PublicConfig.Ipv4, node.NodeID, err.Error())
			continue
		}
		if ip.IsPrivate() {
			log.Printf("public ip %s of node %d is private", node.PublicConfig.Ipv4, node.NodeID)
			continue
		}
		return uint32(node.NodeID), nil
	}
	return 0, errors.New("no nodes with public ipv4")
}
func getNodeFreeWGPort(ctx context.Context, nodeClient *client.NodeClient, nodeId uint32) (int, error) {
	rand.Seed(time.Now().UnixNano())
	freeports, err := nodeClient.NetworkListWGPorts(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list wg ports")
	}
	log.Printf("reserved ports for node %d: %v\n", nodeId, freeports)
	p := uint(rand.Intn(6000) + 2000)

	for isIn(freeports, uint16(p)) {
		p = uint(rand.Intn(6000) + 2000)
	}
	log.Printf("Selected port for node %d is %d\n", nodeId, p)
	return int(p), nil
}

func getNodeEndpoint(ctx context.Context, nodeClient *client.NodeClient) (net.IP, error) {
	publicConfig, err := nodeClient.NetworkGetPublicConfig(ctx)
	log.Printf("publicConfig: %v\n", publicConfig)
	log.Printf("publicConfig.IPv4: %v\n", publicConfig.IPv4)
	log.Printf("publicConfig.IPv.IP: %v\n", publicConfig.IPv4.IP)
	log.Printf("err: %s\n", err)
	if err == nil && publicConfig.IPv4.IP != nil {

		ip := publicConfig.IPv4.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip, nil
		}
	} else if err == nil && publicConfig.IPv6.IP != nil {
		ip := publicConfig.IPv6.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip, nil
		}
	}

	ifs, err := nodeClient.NetworkListInterfaces(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list node interfaces")
	}
	log.Printf("if: %v\n", ifs)

	zosIf, ok := ifs["zos"]
	if !ok {
		return nil, errors.Wrap(ErrNoAccessibleInterfaceFound, "no zos interface")
	}
	for _, ip := range zosIf {
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), ip.IsPrivate())
		if !ip.IsGlobalUnicast() || ip.IsPrivate() {
			continue
		}

		return ip, nil
	}
	return nil, errors.Wrap(ErrNoAccessibleInterfaceFound, "no public ipv4 or ipv6 on zos interface found")
}
