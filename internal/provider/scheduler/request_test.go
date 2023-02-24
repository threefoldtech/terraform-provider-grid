package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/grid_proxy_server/pkg/types"
)

func TestFulfilsSuccess(t *testing.T) {
	cap := freeCapacity(&node)
	nodeInfo := nodeInfo{
		FreeCapacity: &cap,
		Node: types.Node{
			FarmID: 1,
			PublicConfig: types.PublicConfig{
				Ipv4:   "1.2.3.4",
				Domain: "example.com",
			},
		},
	}
	farm := farmInfo{
		freeIPs: 1,
	}
	assert.Equal(t, nodeInfo.fulfils(&Request{
		Capacity: Capacity{
			MRU: 3,
			SRU: 3,
			HRU: 3,
		},
		FarmId:         1,
		PublicIpsCount: 1,
		PublicConfig:   false,
	}, farm), true, "fullfil-success")
}

func TestFulfilsFail(t *testing.T) {
	cap := freeCapacity(&node)
	nodeInfo := nodeInfo{
		FreeCapacity: &cap,
		Node: types.Node{
			FarmID: 1,
			PublicConfig: types.PublicConfig{
				Ipv4:   "",
				Domain: "",
			},
		},
	}
	req := Request{
		Capacity: Capacity{
			MRU: 3,
			SRU: 3,
			HRU: 3,
		},
		FarmId:         1,
		PublicIpsCount: 0,
		PublicConfig:   false,
	}
	farmInfo := farmInfo{
		freeIPs: 1,
	}
	assert.Equal(t, nodeInfo.fulfils(&req, farmInfo), true, "this request should be successful")

	violations := map[string]func(r *Request){
		"mru":              func(r *Request) { r.Capacity.MRU = 4 },
		"sru":              func(r *Request) { r.Capacity.SRU = 9 },
		"hru":              func(r *Request) { r.Capacity.HRU = 4 },
		"farm_id":          func(r *Request) { r.FarmId = 2 },
		"public_ips_count": func(r *Request) { r.PublicIpsCount = 3 },
		"public_config":    func(r *Request) { r.PublicConfig = true },
	}
	for key, fn := range violations {
		cp := req
		fn(&cp)

		assert.Equal(t, nodeInfo.fulfils(&cp, farmInfo), false, fmt.Sprintf("fullfil-fail-%s", key))
	}
}

func TestConstructFilter(t *testing.T) {
	r := Request{
		Capacity: Capacity{
			MRU: 1,
			SRU: 2,
			HRU: 3,
		},
		Name:           "a",
		FarmId:         1,
		PublicIpsCount: 1,
		PublicConfig:   false,
		Certified:      true,
	}

	con := r.constructFilter(1)
	assert.Equal(t, *con.Status, "up", "construct-filter-status")
	assert.Equal(t, *con.FreeMRU, uint64(1), "construct-filter-mru")
	assert.Equal(t, *con.FreeSRU, uint64(2), "construct-filter-sru")
	assert.Equal(t, *con.FreeHRU, uint64(3), "construct-filter-hru")
	assert.Empty(t, con.Country, "construct-filter-country")
	assert.Empty(t, con.City, "construct-filter-city")
	assert.Equal(t, con.FarmIDs, []uint64{uint64(r.FarmId)}, "construct-filter-farm-ids")
	assert.Equal(t, *con.FreeIPs, uint64(1), "construct-filter-free-ips")
	assert.Empty(t, con.IPv4, "construct-filter-ipv4")
	assert.Empty(t, con.IPv6, "construct-filter-ipv6")
	assert.Empty(t, con.Domain, "construct-filter-domain")
	assert.Empty(t, con.Rentable, "construct-filter-rentable")
	assert.Empty(t, con.RentedBy, "construct-filter-rented-by")
	assert.Equal(t, *con.AvailableFor, uint64(1), "construct-filter-available-for")
}
