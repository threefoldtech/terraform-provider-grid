// Package scheduler provides a simple scheduler interface to request deployments on nodes.
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
	Capacity       Capacity
	Name           string
	FarmId         uint32
	PublicConfig   bool
	PublicIpsCount uint32
	Certified      bool
	Dedicated      bool
	NodeExclude    []uint32
}

func (r *Request) constructFilter(twinID uint64) (f proxyTypes.NodeFilter) {
	f.Status = &statusUP
	f.AvailableFor = &twinID
	if r.FarmId != "" {
		f.FarmName = &r.Farm
	}
	if r.Capacity.HRU != 0 {
		f.FreeHRU = &r.Capacity.HRU
	}
	if r.Capacity.SRU != 0 {
		f.FreeSRU = &r.Capacity.SRU
	}
	if r.Capacity.MRU != 0 {
		f.FreeMRU = &r.Capacity.MRU
	}
	if r.HasDomain {
		f.Domain = &trueVal
	}
	if r.HasIPv4 {
		f.IPv4 = &trueVal
	}
	return f
}
