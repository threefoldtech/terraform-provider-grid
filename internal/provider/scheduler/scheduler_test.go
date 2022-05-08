package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/internal/gridproxy"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type GridProxyClientMock struct {
	farms []gridproxy.Farm
	nodes []gridproxy.Node
}

func (m *GridProxyClientMock) Ping() error {
	return nil
}

func (m *GridProxyClientMock) Nodes(filter gridproxy.NodeFilter, pagination gridproxy.Limit) (res []gridproxy.Node, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]gridproxy.Node, 0), nil
	}
	res = m.nodes[start:end]
	return
}

func (m *GridProxyClientMock) Farms(filter gridproxy.FarmFilter, pagination gridproxy.Limit) (res gridproxy.FarmResult, err error) {
	start, end := (pagination.Page-1)*pagination.Size, pagination.Page*pagination.Size
	if int(end) > len(m.nodes) {
		end = uint64(len(m.nodes))
	}
	if end <= start {
		return make([]gridproxy.Farm, 0), nil
	}
	res = m.farms[start:end]
	return
}

func (m *GridProxyClientMock) Node(nodeID uint32) (res gridproxy.NodeInfo, err error) {
	for _, node := range m.nodes {
		if node.NodeID == nodeID {
			res = gridproxy.NodeInfo{
				Capacity: gridproxy.CapacityResult{
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

func (m *GridProxyClientMock) NodeStatus(nodeID uint32) (res gridproxy.NodeStatus, err error) {
	return
}

func (m *GridProxyClientMock) AddFarm(farm gridproxy.Farm) {
	m.farms = append(m.farms, farm)
}

func (m *GridProxyClientMock) AddNode(id uint32, node gridproxy.Node) {
	m.nodes = append(m.nodes, node)
}

func TestSchedulerEmpty(t *testing.T) {
	proxy := &GridProxyClientMock{}
	scheduler := NewScheduler(proxy)
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
	proxy.AddNode(1, gridproxy.Node{
		NodeID: 1,
		TotalResources: gridtypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: gridtypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: gridproxy.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(gridproxy.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    17,
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
	proxy.AddNode(2, gridproxy.Node{})
	proxy.AddNode(3, gridproxy.Node{})
	proxy.AddNode(4, gridproxy.Node{})
	proxy.AddNode(1, gridproxy.Node{
		NodeID: 1,
		TotalResources: gridtypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: gridtypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: gridproxy.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(gridproxy.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    17,
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
	proxy.AddNode(1, gridproxy.Node{
		NodeID: 1,
		TotalResources: gridtypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: gridtypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: gridproxy.PublicConfig{
			Domain: "",
			Ipv4:   "",
			Ipv6:   "",
		},
	})
	proxy.AddFarm(gridproxy.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})

	req := Request{
		Cap: Capacity{
			Hru:    3,
			Sru:    17,
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
		scheduler := NewScheduler(proxy)
		cp := req
		fn(&cp)
		_, err := scheduler.Schedule(&cp)
		assert.Error(t, err, fmt.Sprintf("scheduler-failure-%s", key))
	}
}
func TestSchedulerFailureAfterSuccess(t *testing.T) {
	proxy := &GridProxyClientMock{}
	proxy.AddNode(1, gridproxy.Node{
		NodeID: 1,
		TotalResources: gridtypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: gridtypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: gridproxy.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(gridproxy.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    2,
			Sru:    16,
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
	proxy.AddNode(1, gridproxy.Node{
		NodeID: 1,
		TotalResources: gridtypes.Capacity{
			HRU: 5,
			SRU: 10,
			MRU: 15,
		},
		UsedResources: gridtypes.Capacity{
			HRU: 2,
			SRU: 3,
			MRU: 4,
		},
		FarmID: 1,
		PublicConfig: gridproxy.PublicConfig{
			Domain: "a",
			Ipv4:   "a",
			Ipv6:   "d",
		},
	})
	proxy.AddFarm(gridproxy.Farm{
		Name:   "freefarm",
		FarmID: 1,
	})
	scheduler := NewScheduler(proxy)
	nodeID, err := scheduler.Schedule(&Request{
		Cap: Capacity{
			Hru:    2,
			Sru:    16,
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
