package scheduler

import (
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

var (
	StatusUP = "up"
	trueVal  = true
)

func freeCapacity(node *proxytypes.Node) Capacity {
	var res Capacity

	res.Memory = uint64(node.TotalResources.MRU) - uint64(node.UsedResources.MRU)
	res.Hru = uint64(node.TotalResources.HRU) - uint64(node.UsedResources.HRU)
	res.Sru = uint64(node.TotalResources.SRU) - uint64(node.UsedResources.SRU)

	return res
}

func fullfils(node *nodeInfo, r *Request) bool {
	if r.Cap.Memory > node.FreeCapacity.Memory ||
		r.Cap.Hru > node.FreeCapacity.Hru ||
		r.Cap.Sru > node.FreeCapacity.Sru ||
		(r.farmID != 0 && node.FarmID != r.farmID) ||
		(r.HasDomain && !node.HasDomain) ||
		(r.HasIPv4 && !node.HasIPv4) {
		return false
	}
	return true
}

func subtract(node *Capacity, r *Request) {
	node.Memory -= r.Cap.Memory
	node.Hru -= r.Cap.Hru
	node.Sru -= r.Cap.Sru
}

func constructFilter(r *Request, twinID uint64) (f proxytypes.NodeFilter) {
	f.Status = &StatusUP
	f.AvailableFor = &twinID
	if r.Farm != "" {
		f.FarmName = &r.Farm
	}
	if r.Cap.Hru != 0 {
		f.FreeHRU = &r.Cap.Hru
	}
	if r.Cap.Sru != 0 {
		f.FreeSRU = &r.Cap.Sru
	}
	if r.Cap.Hru != 0 {
		f.FreeMRU = &r.Cap.Memory
	}
	if r.HasDomain {
		f.Domain = &trueVal
	}
	if r.HasIPv4 {
		f.IPv4 = &trueVal
	}
	return f
}
