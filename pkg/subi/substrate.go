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

var (
	_ Substrate = &SubstrateImpl{}
)

type SubstrateImpl struct {
	sub *substrate.Substrate
}

func (s *SubstrateImpl) CancelContract(identity substrate.Identity, contract uint64) error {
	return s.sub.CancelContract(identity, contract)
}

func (s *SubstrateImpl) CancelDeployment(identity substrate.Identity, deploymentID uint64) error {
	return s.sub.CancelDeployment(identity, deploymentID)
}

func (s *SubstrateImpl) Close() {
	s.sub.Close()
}

func (s *SubstrateImpl) CreateCapacityReservationContract(
	identity substrate.Identity,
	farm uint32,
	policy substrate.CapacityReservationPolicy,
	solutionProviderID *uint64,
) (uint64, error) {
	return s.sub.CreateCapacityReservationContract(
		identity,
		farm,
		policy,
		solutionProviderID,
	)
}

func (s *SubstrateImpl) CreateDeployment(
	identity substrate.Identity,
	capacityReservationContractID uint64,
	hash substrate.HexHash,
	data string,
	resources substrate.Resources,
	publicIPs uint32,
) (uint64, error) {
	return s.sub.CreateDeployment(
		identity,
		capacityReservationContractID,
		hash,
		data,
		resources,
		publicIPs,
	)
}

func (s *SubstrateImpl) CreateGroup(identity substrate.Identity) (uint32, error) {
	return s.sub.CreateGroup(identity)
}

func (s *SubstrateImpl) DeleteGroup(identity substrate.Identity, groupID uint32) error {
	return s.sub.DeleteGroup(identity, groupID)
}

func (s *SubstrateImpl) GetAccount(identity substrate.Identity) (info types.AccountInfo, err error) {
	return s.sub.GetAccount(identity)
}

func (s *SubstrateImpl) CreateNameContract(identity substrate.Identity, name string) (uint64, error) {
	return s.sub.CreateNameContract(identity, name)
}

func (s *SubstrateImpl) GetContract(id uint64) (*substrate.Contract, error) {
	return s.sub.GetContract(id)
}

func (s *SubstrateImpl) GetDeployment(id uint64) (*substrate.Deployment, error) {
	return s.sub.GetDeployment(id)
}

func (s *SubstrateImpl) GetGroup(id uint64) (*substrate.Group, error) {
	return s.sub.GetGroup(id)
}

func (s *SubstrateImpl) GetNode(id uint32) (*substrate.Node, error) {
	return s.sub.GetNode(id)
}

func (s *SubstrateImpl) GetTwin(id uint32) (*substrate.Twin, error) {
	return s.sub.GetTwin(id)
}
func (s *SubstrateImpl) GetTwinByPubKey(pk []byte) (uint32, error) {
	return s.sub.GetTwinByPubKey(pk)
}

func (s *SubstrateImpl) UpdateCapacityReservationContract(
	identity substrate.Identity,
	capID uint64,
	resources substrate.Resources,
) error {
	return s.sub.UpdateCapacityReservationContract(
		identity,
		capID,
		resources,
	)
}

func (s *SubstrateImpl) UpdateDeployment(
	identity substrate.Identity,
	id uint64,
	hash substrate.HexHash,
	data string,
	resources *substrate.Resources,
) error {
	return s.sub.UpdateDeployment(
		identity,
		id,
		hash,
		data,
		resources,
	)
}
