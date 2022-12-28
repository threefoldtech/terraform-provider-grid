package client

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type NodeClientGetter interface {
	GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error)
}
type NodeClientPool struct {
	clients map[uint32]*NodeClient
	rmb     rmb.Client
}

func NewNodeClientPool(rmb rmb.Client) *NodeClientPool {
	return &NodeClientPool{
		clients: make(map[uint32]*NodeClient),
		rmb:     rmb,
	}
}

func (p *NodeClientPool) GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error) {
	cl, ok := p.clients[nodeID]
	if ok {
		return cl, nil
	}
	twinID, err := sub.GetNodeTwin(nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node")
	}

	cl = NewNodeClient(uint32(twinID), p.rmb)
	p.clients[nodeID] = cl
	return cl, nil
}
