// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var (
	statusUP = "up"
	trueVal  = true
)

// Request struct for requesting a capacity
type Request struct {
	Capacity       Capacity
	Name           string
	FarmID         uint32
	PublicConfig   bool
	PublicIpsCount uint32
	Certified      bool
	Dedicated      bool
	NodeExclude    []uint32
	Distinct       bool
}

func (r *Request) constructFilter(twinID uint64) (f proxyTypes.NodeFilter) {
	// this filter only lacks certification type, which is validated after.
	// grid proxy should support filtering a node by certification type.
	f.Status = []string{statusUP}
	f.AvailableFor = &twinID
	f.Healthy = &trueVal
	if r.FarmID != 0 {
		f.FarmIDs = []uint64{uint64(r.FarmID)}
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
	if r.PublicConfig {
		f.Domain = &trueVal
	}
	if r.PublicIpsCount != 0 {
		count := uint64(r.PublicIpsCount)
		f.FreeIPs = &count
	}
	if r.Dedicated {
		f.Rentable = &trueVal
	}
	return f
}
