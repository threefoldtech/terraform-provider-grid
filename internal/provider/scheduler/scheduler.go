// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	"fmt"
	"math/rand"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

// nodeInfo related to scheduling
type nodeInfo struct {
	FreeCapacity *Capacity
	Node         proxyTypes.Node
}

type farmInfo struct {
	freeIPs uint64
}

func (s *Scheduler) consumePublicIPs(farmID uint32, IPs uint32) {
	farm := s.farms[farmID]
	farm.freeIPs -= uint64(IPs)
}

func (s *Scheduler) fulfils(node nodeInfo, r *Request) bool {
	if r.Capacity.MRU > node.FreeCapacity.MRU ||
		r.Capacity.HRU > node.FreeCapacity.HRU ||
		r.Capacity.SRU > node.FreeCapacity.SRU ||
		(r.FarmId != 0 && node.Node.FarmID != int(r.FarmId)) ||
		(r.PublicConfig && node.Node.PublicConfig.Domain != "") ||
		(r.PublicIpsCount > uint32(s.farms[uint32(node.Node.FarmID)].freeIPs)) ||
		(r.Dedicated && !node.Node.Dedicated) ||
		contains(r.NodeExclude, uint32(node.Node.NodeID)) {

		return false
	}
	return true
}

// Scheduler struct for scheduling
type Scheduler struct {
	nodes  map[uint32]nodeInfo
	farms  map[uint32]farmInfo
	twinID uint64
	// mapping from farm name to its id
	farmIDS         map[uint32]int
	gridProxyClient proxy.Client
}

// NewScheduler generates a new scheduler
func NewScheduler(gridProxyClient proxy.Client, twinID uint64) Scheduler {
	return Scheduler{
		nodes:           map[uint32]nodeInfo{},
		gridProxyClient: gridProxyClient,

		twinID:  twinID,
		farmIDS: make(map[uint32]int),
		farms:   make(map[uint32]farmInfo),
	}
}

func (n *Scheduler) getFarm(farmID uint32) (farmInfo, error) {
	if f, ok := n.farms[farmID]; ok {
		return f, nil
	}
	id := uint64(farmID)
	farm, _, err := n.gridProxyClient.Farms(proxyTypes.FarmFilter{
		FarmID: &id,
	}, proxyTypes.Limit{
		Size: 1,
		Page: 1,
	})
	if err != nil {
		return farmInfo{}, err
	}
	if len(farm) == 0 {
		return farmInfo{}, fmt.Errorf("farm not found")
	}

	n.farms[farmID] = farmInfo{freeIPs: getPublicIPsCount(farm[0].PublicIps)}
	return n.farms[farmID], nil
}

func getPublicIPsCount(publicIPs []proxyTypes.PublicIP) uint64 {
	freeIPs := 0
	for _, ip := range publicIPs {
		if ip.ContractID == 0 {
			freeIPs++
		}
	}
	return uint64(freeIPs)
}

func (n *Scheduler) getNode(r *Request) uint32 {
	nodes := make([]uint32, 0, len(n.nodes))
	for node := range n.nodes {
		nodes = append(nodes, node)
	}
	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	for _, node := range nodes {
		nodeInfo := n.nodes[node]
		// TODO: later add free ips check when specifying the number of ips is supported
		if n.fulfils(nodeInfo, r) {
			return node
		}
	}
	return 0
}

func (n *Scheduler) addNodes(nodes []proxyTypes.Node) {
	for _, node := range nodes {
		if _, ok := n.nodes[uint32(node.NodeID)]; !ok {
			cap := freeCapacity(&node)
			n.nodes[uint32(node.NodeID)] = nodeInfo{
				FreeCapacity: &cap,
				Node:         node,
			}
		}
	}
}

// Schedule makes sure there's at least one node that satisfies the given request
func (n *Scheduler) Schedule(r *Request) (uint32, error) {
	if true {
		return n.gridProxySchedule(r)
	}
	return n.farmerBotSchedule(r)
}

func (n *Scheduler) gridProxySchedule(r *Request) (uint32, error) {
	f := r.constructFilter(n.twinID)
	l := proxyTypes.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
	}
	_, err := n.getFarm(r.FarmId)
	if err != nil {
		return 0, errors.Wrapf(err, "could not get farm %d", r.FarmId)
	}

	node := n.getNode(r)
	for node == 0 {
		nodes, _, err := n.gridProxyClient.Nodes(f, l)
		if err != nil {
			return 0, errors.Wrap(err, "couldn't list nodes from the grid proxy")
		}
		if len(nodes) == 0 {
			return 0, errors.New("couldn't find a node satisfying the given requirements")
		}
		n.addNodes(nodes)
		node = n.getNode(r)
		if l.Page == 1 && l.Size == 10 {
			l.Page = 2
		} else {
			l.Size *= 2
		}
	}
	n.nodes[node].FreeCapacity.consume(r)
	n.consumePublicIPs(r.FarmId, r.PublicIpsCount)
	return node, nil
}

func (n *Scheduler) farmerBotSchedule(r *Request) (uint32, error) {

	return 0, nil
}

func contains[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if element == e {
			return true
		}
	}
	return false
}
