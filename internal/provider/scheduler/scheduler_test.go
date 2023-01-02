package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

type GridProxyClientMock struct {
	farms []proxyTypes.Farm
	nodes []proxyTypes.Node
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
func TestSchedulerEmpty(t *testing.T) {
	proxy := &GridProxyClientMock{}
	scheduler := NewScheduler(proxy, 1)
	_, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			MRU: 1,
			SRU: 2,
			HRU: 3,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.Error(t, err, "where did you find the node?")
}

func TestSchedulerSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
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
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")
}

func TestSchedulerSuccessOn4thPage(t *testing.T) {
	proxy := &GridProxyClientMock{}
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
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")
}

func TestSchedulerFailure(t *testing.T) {
	proxy := &GridProxyClientMock{}
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
		Cap: Capacity{
			HRU: 3,
			SRU: 7,
			MRU: 11,
		},
		HasIPv4:   false,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: false,
		Certified: false,
	}
	violations := map[string]func(r *Request){
		"mru":    func(r *Request) { r.Cap.MRU = 12 },
		"sru":    func(r *Request) { r.Cap.SRU = 18 },
		"hru":    func(r *Request) { r.Cap.HRU = 4 },
		"ipv4":   func(r *Request) { r.HasIPv4 = true },
		"domain": func(r *Request) { r.HasDomain = true },
	}
	for key, fn := range violations {
		scheduler := NewScheduler(proxy, 1)
		cp := req
		fn(&cp)
		_, err := scheduler.Schedule(&cp)
		assert.Error(t, err, fmt.Sprintf("scheduler-failure-%s", key))
	}
}
func TestSchedulerFailureAfterSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
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
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 2,
			SRU: 6,
			MRU: 10,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

	_, err = scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 1,
			SRU: 1,
			MRU: 2, // this violates
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.Error(t, err, "node would be overloaded")
}

func TestSchedulerSuccessAfterSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
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
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 2,
			SRU: 6,
			MRU: 10,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

	_, err = scheduler.Schedule(&Request{
		Cap: Capacity{
			HRU: 1,
			SRU: 1,
			MRU: 1,
		},
		HasIPv4:   true,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: true,
		Certified: false,
	})
	assert.NoError(t, err, "there's a satisfying node")
	assert.Equal(t, nodeID, uint32(1), "the node id should be 1")

}
