package client

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"

	"github.com/threefoldtech/rmb-sdk-go"
)

// NodeClientGetter is an interface for node client
type NodeClientGetter interface {
	GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error)
}

// NodeClientPool is a pool for node clients and rmb
type NodeClientPool struct {
	clients map[uint32]*NodeClient
	rmb     rmb.Client
	mux     sync.RWMutex
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
	p.mux.RLock()
	cl, ok := p.clients[nodeID]
	p.mux.RUnlock()

	if ok {
		return cl, nil
	}

	twinID, err := sub.GetNodeTwin(nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get node %d", nodeID)
	}

	cl = NewNodeClient(uint32(twinID), p.rmb)

	p.mux.Lock()
	p.clients[nodeID] = cl
	p.mux.Unlock()

	return cl, nil
}
