// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

// NoNodesFoundErr for empty nodes returned from scheduler
var NoNodesFoundErr = errors.New("couldn't find a node satisfying the given requirements")

// Scheduler struct for scheduling
type Scheduler struct {
	nodes           map[uint32]nodeInfo
	farms           map[uint32]farmInfo
	twinID          uint64
	gridProxyClient proxy.Client
	rmbClient       rmb.Client
}

// nodeInfo related to scheduling
type nodeInfo struct {
	FreeCapacity *Capacity
	Node         proxyTypes.Node
}

type farmInfo struct {
	freeIPs           uint64
	certificationType string
	farmerTwinID      uint32
}

func (s *Scheduler) consumePublicIPs(farmID uint32, IPs uint32) {
	farm := s.farms[farmID]
	farm.freeIPs -= uint64(IPs)
}

func (node *nodeInfo) fulfils(r *Request, farm farmInfo) bool {
	if r.Capacity.MRU > node.FreeCapacity.MRU ||
		r.Capacity.HRU > node.FreeCapacity.HRU ||
		r.Capacity.SRU > node.FreeCapacity.SRU ||
		(r.FarmId != 0 && node.Node.FarmID != int(r.FarmId)) ||
		(r.PublicConfig && node.Node.PublicConfig.Domain == "") ||
		(r.PublicIpsCount > uint32(farm.freeIPs)) ||
		(r.Dedicated && !node.Node.Dedicated) ||
		(r.Certified && node.Node.CertificationType != "Certified") ||
		contains(r.NodeExclude, uint32(node.Node.NodeID)) {
		return false
	}
	return true
}

// NewScheduler generates a new scheduler
func NewScheduler(gridProxyClient proxy.Client, twinID uint64, rmbClient rmb.Client) Scheduler {
	return Scheduler{
		nodes:           map[uint32]nodeInfo{},
		gridProxyClient: gridProxyClient,

		twinID:    twinID,
		farms:     make(map[uint32]farmInfo),
		rmbClient: rmbClient,
	}
}

func (n *Scheduler) getFarmInfo(ctx context.Context, farmID uint32) (farmInfo, error) {
	if f, ok := n.farms[farmID]; ok {
		return f, nil
	}
	id := uint64(farmID)
	farm, _, err := n.gridProxyClient.Farms(ctx, proxyTypes.FarmFilter{
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

	n.farms[farmID] = farmInfo{
		freeIPs:           getPublicIPsCount(farm[0].PublicIps),
		certificationType: farm[0].CertificationType,
		farmerTwinID:      uint32(farm[0].TwinID),
	}
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

func (n *Scheduler) getNode(ctx context.Context, r *Request) uint32 {
	nodes := make([]uint32, 0, len(n.nodes))
	for node := range n.nodes {
		nodes = append(nodes, node)
	}
	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	for _, node := range nodes {
		farm, err := n.getFarmInfo(ctx, uint32(n.nodes[node].Node.FarmID))
		if err != nil {
			continue
		}
		nodeInfo := n.nodes[node]
		if nodeInfo.fulfils(r, farm) {
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
func (n *Scheduler) Schedule(ctx context.Context, r *Request) (uint32, error) {
	if r.FarmId != 0 {
		if n.hasFarmerBot(ctx, r.FarmId) {
			return n.farmerBotSchedule(ctx, r)
		}
	}
	return n.gridProxySchedule(ctx, r)
}

func (n *Scheduler) gridProxySchedule(ctx context.Context, r *Request) (uint32, error) {
	f := r.constructFilter(n.twinID)
	l := proxyTypes.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
	}

	node := n.getNode(ctx, r)
	for node == 0 {
		nodes, _, err := n.gridProxyClient.Nodes(ctx, f, l)
		if err != nil {
			return 0, errors.Wrap(err, "couldn't list nodes from the grid proxy")
		}
		if len(nodes) == 0 {
			return 0, NoNodesFoundErr
		}
		n.addNodes(nodes)
		node = n.getNode(ctx, r)
		if l.Page == 1 && l.Size == 10 {
			l.Page = 2
		} else {
			l.Size *= 2
		}
	}
	n.nodes[node].FreeCapacity.consume(r)
	n.consumePublicIPs(uint32(n.nodes[node].Node.FarmID), r.PublicIpsCount)
	return node, nil
}

func (s *Scheduler) ProcessRequests(ctx context.Context, reqs []Request, assignment map[string]uint32) error {
	assignedNodes := []uint32{}
	for _, node := range assignment {
		if !contains(assignedNodes, node) {
			assignedNodes = append(assignedNodes, node)
		}
	}

	for _, r := range reqs {
		if r.Distinct {
			r.NodeExclude = append(r.NodeExclude, assignedNodes...)
		}
		node, err := s.Schedule(ctx, &r)
		if err != nil {
			return errors.Wrapf(err, "couldn't schedule request %s", r.Name)
		}
		assignment[r.Name] = node
		if !contains(assignedNodes, node) {
			assignedNodes = append(assignedNodes, node)
		}
	}
	return nil
}

func contains[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if element == e {
			return true
		}
	}
	return false
}
