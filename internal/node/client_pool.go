package client

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/rmb"
)

var mux sync.RWMutex

// NodeClientGetter is an interface for node client
type NodeClientGetter interface {
	GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error)
}

// NodeClientPool is a pool for node clients and rmb
type NodeClientPool struct {
	clients map[uint32]*NodeClient
	rmb     rmb.Client
}

// NewNodeClientPool generates a new client pool
func NewNodeClientPool(rmb rmb.Client) *NodeClientPool {
	return &NodeClientPool{
		clients: make(map[uint32]*NodeClient),
		rmb:     rmb,
	}
}

// GetNodeClient gets the node client according to node ID
func (p *NodeClientPool) GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error) {
	mux.RLock()
	cl, ok := p.clients[nodeID]
	mux.RUnlock()

	if ok {
		return cl, nil
	}

	twinID, err := sub.GetNodeTwin(nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get node %d", nodeID)
	}

	cl = NewNodeClient(uint32(twinID), p.rmb)

	mux.Lock()
	p.clients[nodeID] = cl
	mux.Unlock()

	return cl, nil
}
