package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

type GridProxyClientMock struct {
	farms []proxytypes.Farm
	nodes []proxytypes.Node
}

func (m *GridProxyClientMock) Ping() error {
	return nil
}

func (m *GridProxyClientMock) Nodes(filter proxytypes.NodeFilter, pagination proxytypes.Limit) (res []proxytypes.Node, totalCount int, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]proxytypes.Node, 0), 0, nil
	}
	res = m.nodes[start:end]
	return
}

func (m *GridProxyClientMock) Farms(filter proxytypes.FarmFilter, pagination proxytypes.Limit) (res []proxytypes.Farm, totalCount int, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]proxytypes.Farm, 0), 0, nil
	}
	res = m.farms[start:end]
	return
}

func (m *GridProxyClientMock) Node(nodeID uint32) (res proxytypes.NodeWithNestedCapacity, err error) {
	for _, node := range m.nodes {
		if uint32(node.NodeID) == nodeID {
			res = proxytypes.NodeWithNestedCapacity{
				Capacity: proxytypes.CapacityResult{
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

func (m *GridProxyClientMock) NodeStatus(nodeID uint32) (res proxytypes.NodeStatus, err error) {
	return
}

func (m *GridProxyClientMock) AddFarm(farm proxytypes.Farm) {
	m.farms = append(m.farms, farm)
}

func (m *GridProxyClientMock) AddNode(id uint32, node proxytypes.Node) {
	m.nodes = append(m.nodes, node)
}
func (m *GridProxyClientMock) Contracts(filter proxytypes.ContractFilter, pagination proxytypes.Limit) (res []proxytypes.Contract, totalCount int, err error) {
	return
}
func (m *GridProxyClientMock) Twins(filter proxytypes.TwinFilter, pagination proxytypes.Limit) (res []proxytypes.Twin, totalCount int, err error) {
	return
}
func (m *GridProxyClientMock) Counters(filter proxytypes.StatsFilter) (res proxytypes.Counters, err error) {
	return
}
func TestSchedulerEmpty(t *testing.T) {
	proxy := &GridProxyClientMock{}
	scheduler := NewScheduler(proxy, 1)
	_, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Memory: 1,
			Sru:    2,
			Hru:    3,
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
	proxy.AddNode(1, proxytypes.Node{
		NodeID: 1,
		TotalResources: proxytypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxytypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxytypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxytypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    7,
			Memory: 11,
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
		proxy.AddNode(i, proxytypes.Node{})
	}
	proxy.AddNode(1, proxytypes.Node{
		NodeID: 1,
		TotalResources: proxytypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxytypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxytypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxytypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    7,
			Memory: 11,
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
	proxy.AddNode(1, proxytypes.Node{
		NodeID: 1,
		TotalResources: proxytypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxytypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxytypes.PublicConfig{
			Domain: "",
			Ipv4:   "",
			Ipv6:   "",
		},
	})
	proxy.AddFarm(proxytypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})

	req := Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    7,
			Memory: 11,
		},
		HasIPv4:   false,
		Name:      "req",
		Farm:      "freefarm",
		HasDomain: false,
		Certified: false,
	}
	violations := map[string]func(r *Request){
		"mru":    func(r *Request) { r.Cap.Memory = 12 },
		"sru":    func(r *Request) { r.Cap.Sru = 18 },
		"hru":    func(r *Request) { r.Cap.Hru = 4 },
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
	proxy.AddNode(1, proxytypes.Node{
		NodeID: 1,
		TotalResources: proxytypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxytypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxytypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxytypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    2,
			Sru:    6,
			Memory: 10,
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
			Hru:    1,
			Sru:    1,
			Memory: 2, // this violates
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
	proxy.AddNode(1, proxytypes.Node{
		NodeID: 1,
		TotalResources: proxytypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: proxytypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: proxytypes.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(proxytypes.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy, 1)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    2,
			Sru:    6,
			Memory: 10,
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
			Hru:    1,
			Sru:    1,
			Memory: 1,
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
