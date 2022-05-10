package scheduler

import (
	"fmt"
	"math/rand"

	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/internal/gridproxy"
)

// NodeInfo related to scheduling
type nodeInfo struct {
	FreeCapacity *Capacity
	FarmID       int
	HasIPv4      bool
	HasDomain    bool
}

type Scheduler struct {
	nodes  map[uint32]nodeInfo
	twinID uint64
	// mapping from farm name to its id
	farmIDS         map[string]int
	gridProxyClient gridproxy.GridProxyClient
}

func NewScheduler(gridProxyClient gridproxy.GridProxyClient, twinID uint64) Scheduler {
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
	farm, err := n.gridProxyClient.Farms(gridproxy.FarmFilter{
		Name: &farmName,
	}, gridproxy.Limit{
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
		cap := n.nodes[node]
		// TODO: later add free ips check when specifying the number of ips is supported
		if fullfils(&cap, r) {
			return node
		}
	}
	return 0
}

func (n *Scheduler) addNodes(nodes []gridproxy.Node) {
	for _, node := range nodes {
		if _, ok := n.nodes[node.NodeID]; !ok {
			cap := freeCapacity(&node)
			n.nodes[node.NodeID] = nodeInfo{
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
	f := constructFilter(r, n.twinID)
	l := gridproxy.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
	}
	if r.Farm != "" {
		id, err := n.getFarmID(r.Farm)
		if err != nil {
			return 0, errors.Wrapf(err, "couldn't get farm %s id", r.Farm)
		}
		r.farmID = id
	}
	node := n.getNode(r)
	for node == 0 {
		nodes, err := n.gridProxyClient.Nodes(f, l)
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
	subtract(n.nodes[node].FreeCapacity, r)
	return node, nil
}
