// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
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
	farmerTwinID      uint32
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

func (n *Scheduler) getFarmInfo(farmID uint32) (farmInfo, error) {
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

func (n *Scheduler) getNode(r *Request) uint32 {
	nodes := make([]uint32, 0, len(n.nodes))
	for node := range n.nodes {
		nodes = append(nodes, node)
	}
	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	for _, node := range nodes {
		farm, err := n.getFarmInfo(r.FarmId)
		if err != nil {
			continue
		}
		nodeInfo := n.nodes[node]
		// TODO: later add free ips check when specifying the number of ips is supported
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
func (n *Scheduler) Schedule(r *Request) (uint32, error) {
	// check if farm id is set
	// if farm id is set, try farmerbot fist, then gridproxy
	// if not, use gridproxy without specifying farm id

	if r.FarmId != 0 {
		if n.hasFarmerBot(r.FarmId) {
			log.Printf("using farmerbot")
			return n.farmerBotSchedule(r)
		}
	}
	log.Printf("using gridproxy")
	return n.gridProxySchedule(r)

}

func (s *Scheduler) hasFarmerBot(farmID uint32) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.getFarmInfo(farmID)
	if err != nil {
		return false
	}
	args := Args{}
	data := s.generateFarmerBotAction(farmID, args, "farmerbot.farmmanager.version")
	log.Printf("ping outgoing data: %+v", data)
	b, err := json.Marshal(data)
	if err != nil {
		log.Printf("marshalling error: %+v", err)
		return false
	}
	dataStr := base64.StdEncoding.EncodeToString(b)
	err = s.rmbClient.Call(ctx, uint32(s.twinID), "farmerbot.farmmanager.version", &dataStr, nil)
	log.Printf("ping error: %+v", err.Error())
	log.Printf("ping incoming data: %+v", data)
	return err == nil
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
	_, err := n.getFarmInfo(r.FarmId)
	if err != nil {
		return 0, errors.Wrap(err, "error getting farm info")
	}
	args := generateFarmerBotArgs(r)
	data := n.generateFarmerBotAction(r.FarmId, args, "armerbot.nodemanager.findnode")
	log.Printf("outgoing data: %+v", data)
	err = n.rmbClient.Call(ctx, uint32(n.twinID), "farmerbot.nodemanager.findnode", &data, nil)
	if err != nil {
		return 0, err
	}
	log.Printf("incoming data: %+v", data)
	return data.Result.Params.Value, nil
}

func generateFarmerBotArgs(r *Request) Args {
	return Args{
		RequiredHRU:  &r.Capacity.HRU,
		RequiredSRU:  &r.Capacity.SRU,
		RequiredMRU:  &r.Capacity.MRU,
		RequiredCRU:  &r.Capacity.CRU,
		NodeExclude:  r.NodeExclude,
		Dedicated:    &r.Dedicated,
		PublicConfig: &r.PublicConfig,
		PublicIPs:    &r.PublicIpsCount,
		Certified:    &r.Certified,
	}
}

type FarmerBotAction struct {
	Guid         string        `json:"guid"`
	TwinID       uint32        `json:"twinid"`
	Action       string        `json:"action"`
	Args         FarmerBotArgs `json:"args"`
	Result       FarmerBotArgs `json:"result"`
	State        string        `json:"state"`
	Start        uint64        `json:"start"`
	End          uint64        `json:"end"`
	GracePeriod  uint32        `json:"grace_period"`
	Error        string        `json:"error"`
	Timeout      uint32        `json:"timeout"`
	SourceTwinID uint32        `json:"src_twinid"`
	SourceAction string        `json:"src_action"`
	Dependencies []string      `json:"dependencies"`
}

type FarmerBotArgs struct {
	Args   Args
	Params Params
}

type Args struct {
	RequiredHRU  *uint64  `json:"required_hru,omitempty"`
	RequiredSRU  *uint64  `json:"required_sru,omitempty"`
	RequiredCRU  *uint64  `json:"required_cru,omitempty"`
	RequiredMRU  *uint64  `json:"required_mru,omitempty"`
	NodeExclude  []uint32 `json:"node_exlude,omitempty"`
	Dedicated    *bool    `json:"dedicated,omitempty"`
	PublicConfig *bool    `json:"public_config,omitempty"`
	PublicIPs    *uint32  `json:"public_ips"`
	Certified    *bool    `json:"certified,omitempty"`
}

type Params struct {
	Key   string `json:"key"`
	Value uint32 `json:"value"`
}

func (s *Scheduler) generateFarmerBotAction(farmID uint32, args Args, action string) FarmerBotAction {
	return FarmerBotAction{
		Guid:   uuid.NewString(),
		TwinID: s.farms[farmID].farmerTwinID,
		Action: action,
		Args: FarmerBotArgs{
			Args:   args,
			Params: Params{},
		},
		Result:       FarmerBotArgs{},
		State:        "init",
		Start:        uint64(time.Now().Second()),
		End:          0,
		GracePeriod:  0,
		Error:        "",
		Timeout:      6000,
		SourceTwinID: uint32(s.twinID),
	}
}

func contains[T comparable](elements []T, element T) bool {
	for _, e := range elements {
		if element == e {
			return true
		}
	}
	return false
}

/*
	- implement ping farmerbot
	- action should be base64 encoded
	-
*/
