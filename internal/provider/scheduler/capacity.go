// Package scheduler provides a simple scheduler interface to request deployments on nodes.
package scheduler

import (
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Capacity struct for capacity (MRU, SRU, HRU)
type Capacity struct {
	MRU uint64
	SRU uint64
	HRU uint64
	CRU uint64
}

func (c *Capacity) consume(r *Request) {
	c.MRU -= r.Capacity.MRU
	c.HRU -= r.Capacity.HRU
	c.SRU -= r.Capacity.SRU
}

func freeCapacity(node *proxyTypes.Node) Capacity {
	var res Capacity

	res.MRU = uint64(node.TotalResources.MRU) - uint64(node.UsedResources.MRU)
	res.HRU = uint64(node.TotalResources.HRU) - uint64(node.UsedResources.HRU)
	res.SRU = uint64(node.TotalResources.SRU) - uint64(node.UsedResources.SRU)
	res.CRU = node.TotalResources.CRU - node.UsedResources.CRU
	return res
}
