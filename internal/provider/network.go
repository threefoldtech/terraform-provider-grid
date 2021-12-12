package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var ErrNoAccessibleInterfaceFound = fmt.Errorf("couldn't find a publicly accessible ipv4 or ipv6")

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

func getPublicNode(ctx context.Context, ncPool NodeClientCollection, graphqlURL string, preferedNodes []uint32) (uint32, error) {
	preferedNodesSet := make(map[uint32]struct{})
	for _, node := range preferedNodes {
		preferedNodesSet[node] = struct{}{}
	}
	client := graphql.NewClient(graphqlURL, nil)
	var q struct {
		Nodes []struct {
			NodeId       graphql.Int
			PublicConfig struct {
				Ipv4 graphql.String
			}
		}
	}
	err := client.Query(ctx, &q, nil)
	if err != nil {
		return 0, err
	}
	preferedFn := func(nodeID uint32) int {
		if _, ok := preferedNodesSet[nodeID]; ok {
			return 0
		} else {
			return 1
		}
	}
	sort.Slice(q.Nodes, func(i, j int) bool {
		return preferedFn(uint32(q.Nodes[i].NodeId)) < preferedFn(uint32(q.Nodes[j].NodeId))
	})
	for _, node := range q.Nodes {
		if node.PublicConfig.Ipv4 != "" {
			log.Printf("found a node with ipv4 public config: %d %s\n", node.NodeId, node.PublicConfig.Ipv4)
			if err := validatePublicNode(ctx, uint32(node.NodeId), ncPool); err != nil {
				log.Printf("error checking public node %d: %s", node.NodeId, err.Error())
				continue
			}
			return uint32(node.NodeId), nil
		}
	}
	return 0, errors.New("no nodes with public ipv4")

}
func validatePublicNode(ctx context.Context, nodeID uint32, ncPool NodeClientCollection) error {
	sub, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	nodeClient, err := ncPool.getNodeClient(nodeID)
	if err != nil {
		return errors.Wrap(err, "couldn't get node client")
	}
	publicConfig, err := nodeClient.NetworkGetPublicConfig(sub)
	if err != nil {
		return errors.Wrap(err, "couldn't get node public config")
	}
	if publicConfig.IPv4.IP == nil {
		return errors.New("node doesn't have a public ip in its config")
	}
	if publicConfig.IPv4.IP.IsPrivate() {
		return errors.New("node has a private ip in its public ip")
	}
	return nil
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

func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{}
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func getNodeEndpoint(ctx context.Context, nodeClient *client.NodeClient) (net.IP, error) {
	publicConfig, err := nodeClient.NetworkGetPublicConfig(ctx)
	log.Printf("publicConfig: %v\n", publicConfig)
	log.Printf("publicConfig.IPv4: %v\n", publicConfig.IPv4)
	log.Printf("publicConfig.IPv.IP: %v\n", publicConfig.IPv4.IP)
	log.Printf("err: %s\n", err)
	if err == nil && publicConfig.IPv4.IP != nil {

		ip := publicConfig.IPv4.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
			return ip, nil
		}
	} else if err == nil && publicConfig.IPv6.IP != nil {
		ip := publicConfig.IPv6.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
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
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if !ip.IsGlobalUnicast() || isPrivateIP(ip) {
			continue
		}

		return ip, nil
	}
	return nil, errors.Wrap(ErrNoAccessibleInterfaceFound, "no public ipv4 or ipv6 on zos interface found")
}
