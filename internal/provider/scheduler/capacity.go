package scheduler

import (
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
)

// Capacity struct for capacity (MRU, SRU, HRU)
type Capacity struct {
	MRU uint64
	SRU uint64
	HRU uint64
}

func (c *Capacity) consume(r *Request) {
	c.MRU -= r.Cap.MRU
	c.HRU -= r.Cap.HRU
	c.SRU -= r.Cap.SRU
}

func freeCapacity(node *proxyTypes.Node) Capacity {
	var res Capacity

	res.MRU = uint64(node.TotalResources.MRU) - uint64(node.UsedResources.MRU)
	res.HRU = uint64(node.TotalResources.HRU) - uint64(node.UsedResources.HRU)
	res.SRU = uint64(node.TotalResources.SRU) - uint64(node.UsedResources.SRU)

	return res
}
