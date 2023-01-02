package scheduler

import (
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

var (
	statusUP = "up"
	trueVal  = true
)

// Request struct for requesting a capacity
type Request struct {
	Cap       Capacity
	Name      string
	Farm      string
	HasIPv4   bool
	HasDomain bool
	Certified bool

	farmID int
}

func (r *Request) constructFilter(twinID uint64) (f proxyTypes.NodeFilter) {
	f.Status = &statusUP
	f.AvailableFor = &twinID
	if r.Farm != "" {
		f.FarmName = &r.Farm
	}
	if r.Cap.HRU != 0 {
		f.FreeHRU = &r.Cap.HRU
	}
	if r.Cap.SRU != 0 {
		f.FreeSRU = &r.Cap.SRU
	}
	if r.Cap.MRU != 0 {
		f.FreeMRU = &r.Cap.MRU
	}
	if r.HasDomain {
		f.Domain = &trueVal
	}
	if r.HasIPv4 {
		f.IPv4 = &trueVal
	}
	return f
}
