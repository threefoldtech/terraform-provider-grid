package subi

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/threefoldtech/substrate-client"
)

type Substrate interface {
	CancelContract(identity substrate.Identity, contract uint64) error
	CancelDeployment(identity substrate.Identity, deploymentID uint64) error
	Close()
	CreateCapacityReservationContract(
		identity substrate.Identity,
		farm uint32,
		policy substrate.CapacityReservationPolicy,
		solutionProviderID *uint64,
	) (uint64, error)
	CreateDeployment(identity substrate.Identity, capacityReservationContractID uint64, hash substrate.HexHash, data string, resources substrate.Resources, publicIPs uint32) (uint64, error)
	CreateNameContract(identity substrate.Identity, name string) (uint64, error)
	CreateGroup(identity substrate.Identity) (uint32, error)
	DeleteGroup(identity substrate.Identity, groupID uint32) error
	GetAccount(identity substrate.Identity) (info types.AccountInfo, err error)
	GetContract(id uint64) (*substrate.Contract, error)
	GetDeployment(id uint64) (*substrate.Deployment, error)
	GetGroup(id uint64) (*substrate.Group, error)
	GetNode(id uint32) (*substrate.Node, error)
	GetTwin(id uint32) (*substrate.Twin, error)
	GetTwinByPubKey(pk []byte) (uint32, error)
	UpdateCapacityReservationContract(identity substrate.Identity, capID uint64, resources substrate.Resources) error
	UpdateDeployment(identity substrate.Identity, id uint64, hash substrate.HexHash, data string, resources *substrate.Resources) error
}
