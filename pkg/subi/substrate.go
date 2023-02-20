// subi package exposes substrate functionality
package subi

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

type Substrate interface {
	CancelContract(identity substrate.Identity, contractID uint64) error
	CreateNodeContract(identity substrate.Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error)
	UpdateNodeContract(identity substrate.Identity, contract uint64, body string, hash string) (uint64, error)
	Close()
	GetTwinByPubKey(pk []byte) (uint32, error)
}
type SubstrateExt interface {
	Substrate
	EnsureContractCanceled(identity substrate.Identity, contractID uint64) error
	DeleteInvalidContracts(contracts map[uint32]uint64) error
	IsValidContract(contractID uint64) (bool, error)
	InvalidateNameContract(
		ctx context.Context,
		identity substrate.Identity,
		contractID uint64,
		name string,
	) (uint64, error)
	GetContract(id uint64) (*substrate.Contract, error)
	GetNodeTwin(id uint32) (uint32, error)
	CreateNameContract(identity substrate.Identity, name string) (uint64, error)
	GetAccount(identity substrate.Identity) (substrate.AccountInfo, error)
	GetContractIDByNameRegistration(name string) (uint64, error)
	GetTwinPK(twinID uint32) ([]byte, error)
	GetTwin(twinID uint32) (*substrate.Twin, error)
}
type SubstrateImpl struct {
	*substrate.Substrate
}

func (s *SubstrateImpl) GetTwin(twinID uint32) (*substrate.Twin, error) {
	return s.Substrate.GetTwin(twinID)
}

func (s *SubstrateImpl) GetContractIDByNameRegistration(name string) (uint64, error) {
	res, err := s.Substrate.GetContractIDByNameRegistration(name)
	return res, err
}

func (s *SubstrateImpl) GetAccount(identity substrate.Identity) (substrate.AccountInfo, error) {
	res, err := s.Substrate.GetAccount(identity)
	return res, err
}
func (s *SubstrateImpl) GetTwinPK(id uint32) ([]byte, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return nil, err
	}
	return twin.Account.PublicKey(), nil
}
func (s *SubstrateImpl) CreateNameContract(identity substrate.Identity, name string) (uint64, error) {
	return s.Substrate.CreateNameContract(identity, name)
}
func (s *SubstrateImpl) GetNodeTwin(id uint32) (uint32, error) {
	node, err := s.Substrate.GetNode(id)
	if err != nil {
		return 0, err
	}
	return uint32(node.TwinID), nil
}
func (s *SubstrateImpl) UpdateNodeContract(identity substrate.Identity, contract uint64, body string, hash string) (uint64, error) {
	res, err := s.Substrate.UpdateNodeContract(identity, contract, body, hash)
	return res, err
}
func (s *SubstrateImpl) CreateNodeContract(identity substrate.Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error) {
	res, err := s.Substrate.CreateNodeContract(identity, node, body, hash, publicIPs, solutionProviderID)
	return res, err
}

func (s *SubstrateImpl) GetContract(contractID uint64) (*substrate.Contract, error) {
	return s.Substrate.GetContract(contractID)
}

func (s *SubstrateImpl) CancelContract(identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	return s.Substrate.CancelContract(identity, contractID)
}
func (s *SubstrateImpl) EnsureContractCanceled(identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	return s.Substrate.CancelContract(identity, contractID)
}

func (s *SubstrateImpl) DeleteInvalidContracts(contracts map[uint32]uint64) error {
	for node, contractID := range contracts {
		valid, err := s.IsValidContract(contractID)
		// TODO: handle pause
		if err != nil {
			return err
		}
		if !valid {
			delete(contracts, node)
		}
	}
	return nil
}

func (s *SubstrateImpl) IsValidContract(contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	// TODO: handle pause
	if errors.Is(err, substrate.ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}

func (s *SubstrateImpl) InvalidateNameContract(
	ctx context.Context,
	identity substrate.Identity,
	contractID uint64,
	name string,
) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	if errors.Is(err, substrate.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get name contract info")
	}
	// TODO: paused?
	if !contract.State.IsCreated {
		return 0, nil
	}
	if contract.ContractType.NameContract.Name != name {
		err := s.Substrate.CancelContract(identity, contractID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to cleanup unmatching name contract")
		}
		return 0, nil
	}

	return contractID, nil
}
