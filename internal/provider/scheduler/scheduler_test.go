package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type GridProxyClientMock struct {
	farms []proxyTypes.Farm
	nodes []proxyTypes.Node
}

type RMBClientMock struct {
	nodeID       uint32
	hasFarmerBot bool
}

func (r *RMBClientMock) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	d := data.(FarmerBotAction)
	switch d.Action {
	case FarmerBotVersionAction:
		if r.hasFarmerBot {
			return nil
		}
		return errors.New("this farm does not have a farmer bot")
	case FarmerBotFindNodeAction:
		if r.nodeID == 0 {
			d.Error = "could not find node"
			return nil
		}
		output := result.(*FarmerBotAction)

		output.Result.Params = append(output.Args.Params, Params{Key: "nodeid", Value: strconv.FormatUint(uint64(r.nodeID), 10)})
		return nil
	default:
		return fmt.Errorf("fn: %s not supported", d.Action)
	}
}

func (m *GridProxyClientMock) Ping() error {
	return nil
}

func (m *GridProxyClientMock) Nodes(filter proxyTypes.NodeFilter, pagination proxyTypes.Limit) (res []proxyTypes.Node, totalCount int, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]proxyTypes.Node, 0), 0, nil
	}
	res = m.nodes[start:end]
	return
}

func (m *GridProxyClientMock) Farms(filter proxyTypes.FarmFilter, pagination proxyTypes.Limit) (res []proxyTypes.Farm, totalCount int, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]proxyTypes.Farm, 0), 0, nil
	}
	res = m.farms[start:end]
	return
}

func (m *GridProxyClientMock) Node(nodeID uint32) (res proxyTypes.NodeWithNestedCapacity, err error) {
	for _, node := range m.nodes {
		if uint32(node.NodeID) == nodeID {
			res = proxyTypes.NodeWithNestedCapacity{
				Capacity: proxyTypes.CapacityResult{
					Total: node.TotalResources,
					Used:  node.UsedResources,
				},
			}
			return
		}
	}
	err = fmt.Errorf("node not found")
	return
}

func (m *GridProxyClientMock) NodeStatus(nodeID uint32) (res proxyTypes.NodeStatus, err error) {
	return
}

func (m *GridProxyClientMock) AddFarm(farm proxyTypes.Farm) {
	m.farms = append(m.farms, farm)
}

func (m *GridProxyClientMock) AddNode(id uint32, node proxyTypes.Node) {
	m.nodes = append(m.nodes, node)
}
func (m *GridProxyClientMock) Contracts(filter proxyTypes.ContractFilter, pagination proxyTypes.Limit) (res []proxyTypes.Contract, totalCount int, err error) {
	return
}
func (m *GridProxyClientMock) Twins(filter proxyTypes.TwinFilter, pagination proxyTypes.Limit) (res []proxyTypes.Twin, totalCount int, err error) {
	return
}
func (m *GridProxyClientMock) Counters(filter proxyTypes.StatsFilter) (res proxyTypes.Counters, err error) {
	return
}
func (m *GridProxyClientMock) Contract(contractID uint32) (res proxyTypes.Contract, err error) {
	return
}
func (m *GridProxyClientMock) ContractBills(contractID uint32, limit proxyTypes.Limit) (res []proxyTypes.ContractBilling, count uint, err error) {
	return
}

func TestSchedulerEmpty(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: false,
	}
	scheduler := NewScheduler(proxy, 1, rmbClient)
	_, err := scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			MRU: 1,
			SRU: 2,
			HRU: 3,
		},
		PublicIpsCount: 1,
		Name:           "req",
		FarmId:         10,
		PublicConfig:   true,
		Certified:      false,
		Dedicated:      false,
		NodeExclude:    []uint32{1, 2},
	})
	assert.Error(t, err, "where did you find the node?")
}

func TestSchedulerSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		nodeID:       1,
		hasFarmerBot: true,
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		TotalResources: proxyTypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxyTypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxyTypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxyTypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
		PublicIps: []proxyTypes.PublicIP{
			{
				IP: "asdf",
			},
		},
	})
	scheduler := NewScheduler(proxy, 1, rmbClient)
	nodeID, err := scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		Name:           "req",
		FarmId:         1,
		PublicConfig:   true,
		PublicIpsCount: 1,
		Certified:      false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")
}

func TestSchedulerSuccessOn4thPage(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: true,
		nodeID:       1,
	}
	for i := uint32(2); i <= 30; i++ {
		proxy.AddNode(i, proxyTypes.Node{})
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		TotalResources: proxyTypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxyTypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxyTypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxyTypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
		PublicIps: []proxyTypes.PublicIP{
			{
				IP: "a",
			},
		},
	})
	scheduler := NewScheduler(proxy, 1, rmbClient)
	nodeID, err := scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		PublicConfig:   true,
		Name:           "req",
		FarmId:         1,
		PublicIpsCount: 1,
		Certified:      false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")
}

func TestSchedulerFailure(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		TotalResources: proxyTypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxyTypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxyTypes.PublicConfig{
			Domain: "",
			Ipv4:   "",
			Ipv6:   "",
		},
	})
	proxy.AddFarm(proxyTypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})

	req := Request{
		Capacity: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		PublicIpsCount: 0,
		Name:           "req",
		PublicConfig:   false,
		Certified:      false,
	}
	violations := map[string]func(r *Request){
		"mru":    func(r *Request) { r.Capacity.MRU = 12 },
		"sru":    func(r *Request) { r.Capacity.SRU = 18 },
		"hru":    func(r *Request) { r.Capacity.HRU = 4 },
		"ipv4":   func(r *Request) { r.PublicIpsCount = 15 },
		"domain": func(r *Request) { r.PublicConfig = true },
	}
	for key, fn := range violations {
		scheduler := NewScheduler(proxy, 1, rmbClient)
		cp := req
		fn(&cp)
		_, err := scheduler.Schedule(context.Background(), &cp)
		assert.Error(t, err, fmt.Sprintf("scheduler-failure-%s", key))
	}
}
func TestSchedulerFailureAfterSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: false,
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		TotalResources: proxyTypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxyTypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxyTypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxyTypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
		PublicIps: []proxyTypes.PublicIP{
			{
				IP: "a",
			},
		},
	})
	scheduler := NewScheduler(proxy, 1, rmbClient)
	nodeID, err := scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 2,
			SRU: 6,
			MRU: 10,
		},
		PublicIpsCount: 1,
		Name:           "req",
		FarmId:         1,
		PublicConfig:   true,
		Certified:      false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

	_, err = scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 1,
			SRU: 1,
			MRU: 2, // this violates
		},
		PublicIpsCount: 1,
		Name:           "req",
		FarmId:         1,
		PublicConfig:   true,
		Certified:      false,
	})
	assert.Error(t, err, "node would be overloaded")
}

func TestSchedulerSuccessAfterSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: true,
		nodeID:       1,
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		TotalResources: proxyTypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxyTypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxyTypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxyTypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
		PublicIps: []proxyTypes.PublicIP{
			{
				IP: "a",
			},
		},
	})
	scheduler := NewScheduler(proxy, 1, rmbClient)
	nodeID, err := scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 2,
			SRU: 6,
			MRU: 10,
		},
		PublicIpsCount: 1,
		Name:           "req",
		FarmId:         1,
		PublicConfig:   true,
		Certified:      false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

	_, err = scheduler.Schedule(context.Background(), &Request{
		Capacity: Capacity{
			HRU: 1,
			SRU: 1,
			MRU: 1,
		},
		PublicIpsCount: 1,
		Name:           "req",
		FarmId:         1,
		PublicConfig:   true,
		Certified:      false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

}

func TestExcludingNodes(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: false,
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		FarmID: 1,
	})
	proxy.AddNode(2, proxyTypes.Node{
		NodeID: 2,
		FarmID: 1,
	})
	proxy.AddFarm(proxyTypes.Farm{
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy, 1, rmbClient)
	node, err := scheduler.Schedule(context.Background(), &Request{
		NodeExclude: []uint32{1},
	})
	assert.NoError(t, err)
	assert.Equal(t, node, uint32(2))

	node, err = scheduler.Schedule(context.Background(), &Request{
		NodeExclude: []uint32{2},
	})
	assert.NoError(t, err)
	assert.Equal(t, node, uint32(1))
}

func TestDistinctNodes(t *testing.T) {
	proxy := &GridProxyClientMock{}
	rmbClient := &RMBClientMock{
		hasFarmerBot: false,
	}
	proxy.AddNode(1, proxyTypes.Node{
		NodeID: 1,
		FarmID: 1,
	})
	proxy.AddNode(2, proxyTypes.Node{
		NodeID: 2,
		FarmID: 1,
	})
	proxy.AddNode(3, proxyTypes.Node{
		NodeID: 3,
		FarmID: 1,
	})
	proxy.AddFarm(proxyTypes.Farm{
		FarmID: 1,
	})
	requests := []Request{
		{
			Name:     "r1",
			Distinct: true,
		},
		{
			Name:     "r2",
			Distinct: true,
		},
		{
			Name:     "r3",
			Distinct: true,
		},
	}
	scheduler := NewScheduler(proxy, 1, rmbClient)
	assignment := map[string]uint32{}
	err := scheduler.ProcessRequests(context.Background(), requests, assignment)
	assert.NoError(t, err)
	assert.NotEqual(t, assignment["r1"], assignment["r2"])
	assert.NotEqual(t, assignment["r1"], assignment["r3"])
	assert.NotEqual(t, assignment["r2"], assignment["r3"])
}
