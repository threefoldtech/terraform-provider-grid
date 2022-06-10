package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

var (
	node = proxytypes.Node{
		UsedResources: proxytypes.Capacity{
			HRU: 1,
			SRU: 2,
			MRU: 3,
		},
		TotalResources: proxytypes.Capacity{
			HRU: 4,
			SRU: 5,
			MRU: 6,
		},
	}
)

func TestFreeCapacity(t *testing.T) {
	cap := freeCapacity(&node)
	assert.Equal(t, cap.Hru, uint64(3), "hru")
	assert.Equal(t, cap.Sru, uint64(3), "sru")
	assert.Equal(t, cap.Memory, uint64(3), "mru")
}

func TestFullfilsSuccess(t *testing.T) {
	cap := freeCapacity(&node)
	nodeInfo := nodeInfo{
		FreeCapacity: &cap,
		FarmID:       1,
		HasIPv4:      true,
		HasDomain:    true,
	}
	assert.Equal(t, fullfils(&nodeInfo, &Request{
		Cap: Capacity{
			Memory: 3,
			Sru:    3,
			Hru:    3,
		},
		farmID:    1,
		HasIPv4:   true,
		HasDomain: false,
	}), true, "fullfil-success")
}

func TestFullfilsFail(t *testing.T) {
	cap := freeCapacity(&node)
	nodeInfo := nodeInfo{
		FreeCapacity: &cap,
		FarmID:       1,
		HasIPv4:      false,
		HasDomain:    false,
	}

	req := Request{
		Cap: Capacity{
			Memory: 3,
			Sru:    8,
			Hru:    3,
		},
		farmID:    1,
		HasIPv4:   false,
		HasDomain: false,
	}
	violations := map[string]func(r *Request){
		"mru":     func(r *Request) { r.Cap.Memory = 4 },
		"sru":     func(r *Request) { r.Cap.Sru = 9 },
		"hru":     func(r *Request) { r.Cap.Hru = 4 },
		"farm_id": func(r *Request) { r.farmID = 2 },
		"ipv4":    func(r *Request) { r.HasIPv4 = true },
		"domain":  func(r *Request) { r.HasDomain = true },
	}
	for key, fn := range violations {
		cp := req
		fn(&cp)
		assert.Equal(t, fullfils(&nodeInfo, &cp), false, fmt.Sprintf("fullfil-fail-%s", key))
	}
}

func TestConstructFilter(t *testing.T) {
	var farm string = "freefarm"
	r := Request{
		Cap: Capacity{
			Memory: 1,
			Sru:    2,
			Hru:    3,
		},
		Name:      "a",
		Farm:      farm,
		HasIPv4:   true,
		HasDomain: false,
		Certified: true,
	}

	con := constructFilter(&r, 1)
	assert.Equal(t, *con.Status, "up", "construct-filter-status")
	assert.Equal(t, *con.FreeMRU, uint64(1), "construct-filter-mru")
	assert.Equal(t, *con.FreeSRU, uint64(2), "construct-filter-sru")
	assert.Equal(t, *con.FreeHRU, uint64(3), "construct-filter-hru")
	assert.Empty(t, con.Country, "construct-filter-country")
	assert.Empty(t, con.City, "construct-filter-city")
	assert.Equal(t, *con.FarmName, "freefarm", "construct-filter-farm-name")
	assert.Empty(t, con.FarmIDs, "construct-filter-farm-ids")
	assert.Empty(t, con.FreeIPs, "construct-filter-free-ips")
	assert.Equal(t, *con.IPv4, true, "construct-filter-ipv4")
	assert.Empty(t, con.IPv6, "construct-filter-ipv6")
	assert.Empty(t, con.Domain, "construct-filter-domain")
	assert.Empty(t, con.Rentable, "construct-filter-rentable")
	assert.Empty(t, con.RentedBy, "construct-filter-rented-by")
	assert.Equal(t, *con.AvailableFor, uint64(1), "construct-filter-available-for")
}
