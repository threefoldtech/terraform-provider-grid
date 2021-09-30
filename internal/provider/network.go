package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
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

func getPublicNode(graphqlURL string, preferedNodes []uint32) (uint32, error) {

	client := graphql.NewClient(graphqlURL, nil)
	var q struct {
		Nodes []struct {
			NodeId       graphql.Int
			PublicConfig struct {
				Ipv4 graphql.String
			}
		}
	}
	err := client.Query(context.Background(), &q, nil)
	if err != nil {
		panic(err)
	}
	publicNode := uint32(0)
	for _, node := range q.Nodes {
		if node.PublicConfig.Ipv4 != "" {
			ip, err := gridtypes.ParseIPNet(string(node.PublicConfig.Ipv4))
			if err != nil {
				log.Printf("couldn't parse ip %s: %s", node.PublicConfig.Ipv4, err)
				continue
			}
			if ip.IP.To4().IsPrivate() {
				continue
			}
			if isInUint32(preferedNodes, uint32(node.NodeId)) {
				return uint32(node.NodeId), nil
			} else {
				publicNode = uint32(node.NodeId)
			}
		}
	}
	if publicNode == 0 {
		return 0, errors.New("no nodes with public ipv4")
	} else {
		return publicNode, nil
	}
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

func getNodeEndpoint(ctx context.Context, nodeClient *client.NodeClient) (string, error) {
	publicConfig, err := nodeClient.NetworkGetPublicConfig(ctx)
	log.Printf("publicConfig: %v\n", publicConfig)
	log.Printf("publicConfig.IPv4: %v\n", publicConfig.IPv4)
	log.Printf("publicConfig.IPv.IP: %v\n", publicConfig.IPv4.IP)
	log.Printf("err: %s\n", err)
	if err == nil && publicConfig.IPv4.IP != nil {

		ip := publicConfig.IPv4.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
			return ip.String(), nil
		}
	} else if err == nil && publicConfig.IPv6.IP != nil {
		ip := publicConfig.IPv6.IP
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
			return fmt.Sprintf("[%s]", ip.String()), nil
		}
	}

	ifs, err := nodeClient.NetworkListInterfaces(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't list node interfaces")
	}
	log.Printf("if: %v\n", ifs)

	zosIf, ok := ifs["zos"]
	if !ok {
		return "", errors.New("node doesn't contain zos interface or public config")
	}
	for _, ip := range zosIf {
		log.Printf("ip: %s, globalunicast: %t, privateIP: %t\n", ip.String(), ip.IsGlobalUnicast(), isPrivateIP(ip))
		if !ip.IsGlobalUnicast() || isPrivateIP(ip) {
			continue
		}

		if ip.To4() != nil {
			return ip.String(), nil
		} else {
			return fmt.Sprintf("[%s]", ip.String()), nil
		}
	}
	return "", errors.New("no public config found, and no public ipv4 or ipv6 on zos interface found")
}
