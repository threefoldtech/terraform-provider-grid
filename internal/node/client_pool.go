package client

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type NodeClientCollection interface {
	GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error)
}
type NodeClientPool struct {
	nodeClients map[uint32]*NodeClient
	rmb         rmb.Client
}

func NewNodeClientPool(rmb rmb.Client) *NodeClientPool {
	return &NodeClientPool{
		nodeClients: make(map[uint32]*NodeClient),
		rmb:         rmb,
	}
}

func (k *NodeClientPool) GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*NodeClient, error) {
	cl, ok := k.nodeClients[nodeID]
	if ok {
		return cl, nil
	}
	twinID, err := sub.GetNodeTwin(nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node")
	}

	cl = NewNodeClient(uint32(twinID), k.rmb)
	k.nodeClients[nodeID] = cl
	return cl, nil
}
