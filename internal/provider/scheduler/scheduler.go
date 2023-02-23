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
	FarmID       int
	HasIPv4      bool
	HasDomain    bool
}

func (node *nodeInfo) fulfils(r *Request) bool {
	if r.Capacity.MRU > node.FreeCapacity.MRU ||
		r.Capacity.HRU > node.FreeCapacity.HRU ||
		r.Capacity.SRU > node.FreeCapacity.SRU ||
		(r.farmID != 0 && node.FarmID != r.farmID) ||
		(r.HasDomain && !node.HasDomain) ||
		(r.HasIPv4 && !node.HasIPv4) {
		return false
	}
	return true
}

// Scheduler struct for scheduling
type Scheduler struct {
	nodes  map[uint32]nodeInfo
	twinID uint64
	// mapping from farm name to its id
	farmIDS         map[string]int
	gridProxyClient proxy.Client
}

// NewScheduler generates a new scheduler
func NewScheduler(gridProxyClient proxy.Client, twinID uint64) Scheduler {
	return Scheduler{
		nodes:           map[uint32]nodeInfo{},
		gridProxyClient: gridProxyClient,

		twinID:  twinID,
		farmIDS: make(map[string]int),
	}
}

func (n *Scheduler) getFarmID(farmName string) (int, error) {
	if id, ok := n.farmIDS[farmName]; ok {
		return id, nil
	}
	farm, _, err := n.gridProxyClient.Farms(proxyTypes.FarmFilter{
		Name: &farmName,
	}, proxyTypes.Limit{
		Size: 1,
		Page: 1,
	})
	if err != nil {
		return 0, err
	}
	if len(farm) == 0 {
		return 0, fmt.Errorf("farm not found")
	}
	n.farmIDS[farmName] = farm[0].FarmID
	return farm[0].FarmID, nil
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
		if nodeInfo.fulfils(r) {
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
				HasIPv4:      node.PublicConfig.Ipv4 != "",
				HasDomain:    node.PublicConfig.Domain != "",
				FarmID:       node.FarmID,
			}
		}
	}
}

// Schedule makes sure there's at least one node that satisfies the given request
func (n *Scheduler) Schedule(r *Request) (uint32, error) {
	if true {
		return n.farmerBotSchedule(r)
	}
	return n.gridProxySchedule(r)
}

func (n *Scheduler) gridProxySchedule(r *Request) (uint32, error) {
	f := r.constructFilter(n.twinID)
	l := proxyTypes.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
	}
	if r.FarmId != "" {
		id, err := n.getFarmID(r.Farm)
		if err != nil {
			return 0, errors.Wrapf(err, "couldn't get farm %s id", r.Farm)
		}
		r.farmID = id
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
	return node, nil
}

func (n *Scheduler) farmerBotSchedule(r *Request) (uint32, error) {

	return 0, nil
}
