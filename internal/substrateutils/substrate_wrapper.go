package substrateutils

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/substrate-client"
)

type SubstrateManager interface {
	substrate.Manager
	GetContract(id uint64) (*substrate.Contract, error)
	CancelContract(identity substrate.Identity, contract uint64) error
	CreateNodeContract(identity substrate.Identity, node uint32, body []byte, hash string, publicIPs uint32) (uint64, error)
	UpdateNodeContract(identity substrate.Identity, contract uint64, body []byte, hash string) (uint64, error)
	GetNode(id uint32) (*substrate.Node, error)
	GetTwinByPubKey(pk []byte) (uint32, error)
	GetContractIDByNameRegistration(name string) (uint64, error)
	CreateNameContract(identity substrate.Identity, name string) (uint64, error)
	GetAccount(identity substrate.Identity) (info types.AccountInfo, err error)
	GetTwin(id uint32) (*substrate.Twin, error)
}

type ManagerWrapper struct {
	substrate.Manager
}

func NewManagerWrapper(manager substrate.Manager) *ManagerWrapper {
	return &ManagerWrapper{manager}
}
func (m *ManagerWrapper) GetContract(id uint64) (*substrate.Contract, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetContract(id)
}
func (m *ManagerWrapper) CancelContract(identity substrate.Identity, contract uint64) error {
	substrate, err := m.Substrate()
	if err != nil {
		return errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.CancelContract(identity, contract)
}
func (m *ManagerWrapper) CreateNodeContract(identity substrate.Identity, node uint32, body []byte, hash string, publicIPs uint32) (uint64, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.CreateNodeContract(identity, node, body, hash, publicIPs)
}
func (m *ManagerWrapper) UpdateNodeContract(identity substrate.Identity, contract uint64, body []byte, hash string) (uint64, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.UpdateNodeContract(identity, contract, body, hash)
}
func (m *ManagerWrapper) GetNode(id uint32) (*substrate.Node, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetNode(id)
}
func (m *ManagerWrapper) GetTwinByPubKey(pk []byte) (uint32, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetTwinByPubKey(pk)
}
func (m *ManagerWrapper) GetContractIDByNameRegistration(name string) (uint64, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetContractIDByNameRegistration(name)
}
func (m *ManagerWrapper) CreateNameContract(identity substrate.Identity, name string) (uint64, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.CreateNameContract(identity, name)
}
func (m *ManagerWrapper) GetAccount(identity substrate.Identity) (info types.AccountInfo, err error) {
	substrate, err := m.Substrate()
	if err != nil {
		return info, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetAccount(identity)
}
func (m *ManagerWrapper) GetTwin(id uint32) (*substrate.Twin, error) {
	substrate, err := m.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get substrate client")
	}
	return substrate.GetTwin(id)
}
