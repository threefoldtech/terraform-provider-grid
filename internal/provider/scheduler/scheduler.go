// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/rmb-sdk-go"
)

// nodeInfo related to scheduling
type nodeInfo struct {
	FreeCapacity *Capacity
	Node         proxyTypes.Node
}

type farmInfo struct {
	freeIPs           uint64
	certificationType string
}

func (s *Scheduler) consumePublicIPs(farmID uint32, IPs uint32) {
	farm := s.farms[farmID]
	farm.freeIPs -= uint64(IPs)
}

func (node *nodeInfo) fulfils(r *Request, farm farmInfo) bool {
	log.Printf("request: %+v\nfarminfo: %+v", r, farm)
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

// Scheduler struct for scheduling
type Scheduler struct {
	nodes  map[uint32]nodeInfo
	farms  map[uint32]farmInfo
	twinID uint64
	// mapping from farm name to its id
	farmIDS         map[uint32]int
	gridProxyClient proxy.Client
	rmbClient       rmb.Client
}

// NewScheduler generates a new scheduler
func NewScheduler(gridProxyClient proxy.Client, twinID uint64, rmbClient rmb.Client) Scheduler {
	return Scheduler{
		nodes:           map[uint32]nodeInfo{},
		gridProxyClient: gridProxyClient,

		twinID:    twinID,
		farmIDS:   make(map[uint32]int),
		farms:     make(map[uint32]farmInfo),
		rmbClient: rmbClient,
	}
}

// func (n *Scheduler) getFarmInfo(farmID uint32) (farmInfo, error) {
// 	if f, ok := n.farms[farmID]; ok {
// 		return f, nil
// 	}
// 	id := uint64(farmID)
// 	farm, _, err := n.gridProxyClient.Farms(proxyTypes.FarmFilter{
// 		FarmID: &id,
// 	}, proxyTypes.Limit{
// 		Size: 1,
// 		Page: 1,
// 	})
// 	if err != nil {
// 		return farmInfo{}, err
// 	}
// 	if len(farm) == 0 {
// 		return farmInfo{}, fmt.Errorf("farm not found")
// 	}

// 	n.farms[farmID] = farmInfo{freeIPs: getPublicIPsCount(farm[0].PublicIps)}
// 	return n.farms[farmID], nil
// }

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
		if nodeInfo.fulfils(r, n.farms[r.FarmId]) {
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
	// check if farm id is set
	// if not, send a request to gridproxy for eligible farms
	// process returned farms in batches

	farmFilter := r.constructFarmFilter()
	limit := proxyTypes.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
	}

	for r.FarmId == 0 {
		farms, _, err := n.gridProxyClient.Farms(farmFilter, limit)
		if err != nil {
			return 0, errors.Wrap(err, "could not list available farms from grid proxy")
		}

		if len(farms) == 0 {
			return 0, errors.Wrap(err, "could not find an eligible farm from grid proxy")
		}
		n.addFarms(farms)
		for _, farm := range farms {
			r.FarmId = uint32(farm.FarmID)
			var node uint32
			var err error
			if n.hasFarmerBot(r.FarmId) {
				node, err = n.gridProxySchedule(r)
			} else {
				node, err = n.farmerBotSchedule(r)
			}

			if node != 0 && err == nil {
				return node, nil
			}
		}
		r.FarmId = 0
	}
	return 0, errors.New("could not find an eligible node")

}

func (s *Scheduler) addFarms(farms []proxyTypes.Farm) {
	for _, farm := range farms {
		if _, ok := s.farms[uint32(farm.FarmID)]; ok {
			continue
		}

		s.farms[uint32(farm.FarmID)] = farmInfo{
			freeIPs:           getPublicIPsCount(farm.PublicIps),
			certificationType: farm.CertificationType,
		}
	}
}

func (s *Scheduler) hasFarmerBot(farmID uint32) bool {
	return false
}

func (n *Scheduler) gridProxySchedule(r *Request) (uint32, error) {
	f := r.constructFilter(n.twinID)
	l := proxyTypes.Limit{
		Size:     10,
		Page:     1,
		RetCount: false,
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
	// we need to check if farm id is specified or not
	// if specified, then only this farm will receive a call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data := generateFarmerBotAction(r)
	var res uint32
	err := n.rmbClient.Call(ctx, uint32(n.twinID), "farmerbot.nodemanager.find", data, &res)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func generateFarmerBotAction(r *Request) interface{} {
	return struct {
		Guid   string `json:"guid"`
		TwinID uint32 `json:"twinid"`
		Action string `json:"action"`
	}{}
}

/*
pub struct ActionJobPublic {
pub mut:
	guid         string
	twinid		 u32    	//twinid of the farmerbot
	action   	 string 	//farmerbot.*
	args       	 params.Params
	result       params.Params
	state        string
	start        i64		//epoch
	end          i64		//epoch
	grace_period u32 		//wait till next run, in seconds
	error        string		//string description of what went wrong
	timeout      u32 		//time in seconds, 2h is maximum
	src_twinid	 u32    	//which twin was sending the job, 0 if local
	src_action   string		//unique actor path, runs on top of twin
	dependencies []string	//list of guids we need to wait on
}
*/

func contains[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if element == e {
			return true
		}
	}
	return false
}
