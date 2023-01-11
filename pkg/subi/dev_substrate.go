package subi

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	subdev "github.com/threefoldtech/substrate-client-dev"
)

// Substrate interface for substrate client
type Substrate interface {
	CancelContract(identity Identity, contractID uint64) error
	CreateNodeContract(identity Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error)
	UpdateNodeContract(identity Identity, contract uint64, body string, hash string) (uint64, error)
	Close()
	GetTwinByPubKey(pk []byte) (uint32, error)
}

// SubstrateExt interface for substrate executable functions
type SubstrateExt interface {
	Substrate
	EnsureContractCanceled(identity Identity, contractID uint64) error
	DeleteInvalidContracts(contracts map[uint32]uint64) error
	IsValidContract(contractID uint64) (bool, error)
	InvalidateNameContract(
		ctx context.Context,
		identity Identity,
		contractID uint64,
		name string,
	) (uint64, error)
	GetContract(id uint64) (Contract, error)
	GetNodeTwin(id uint32) (uint32, error)
	CreateNameContract(identity Identity, name string) (uint64, error)
	GetAccount(identity Identity) (types.AccountInfo, error)
	GetTwinIP(twinID uint32) (string, error)
	GetContractIDByNameRegistration(name string) (uint64, error)
	GetTwinPK(twinID uint32) ([]byte, error)
}

// SubstrateDevImpl struct to use dev substrate
type SubstrateDevImpl struct {
	*subdev.Substrate
}

// GetContractIDByNameRegistration returns contract ID using its name
func (s *SubstrateDevImpl) GetContractIDByNameRegistration(name string) (uint64, error) {
	res, err := s.Substrate.GetContractIDByNameRegistration(name)
	return res, normalizeNotFoundErrors(err)
}

// GetTwinIP returns twin IP given its ID
func (s *SubstrateDevImpl) GetTwinIP(id uint32) (string, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return "", normalizeNotFoundErrors(err)
	}
	return twin.IP, nil
}

// GetAccount returns the user's account
func (s *SubstrateDevImpl) GetAccount(identity Identity) (types.AccountInfo, error) {
	res, err := s.Substrate.GetAccount(identity)
	return res, normalizeNotFoundErrors(err)
}

// GetTwinPK returns twin's public key
func (s *SubstrateDevImpl) GetTwinPK(id uint32) ([]byte, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return nil, normalizeNotFoundErrors(err)
	}
	return twin.Account.PublicKey(), nil
}

// GetTwinByPubKey returns the twin's ID using user's public key
func (s *SubstrateDevImpl) GetTwinByPubKey(pk []byte) (uint32, error) {
	res, err := s.Substrate.GetTwinByPubKey(pk)
	return res, normalizeNotFoundErrors(err)
}

// CreateNameContract creates a new name contract
func (s *SubstrateDevImpl) CreateNameContract(identity Identity, name string) (uint64, error) {
	return s.Substrate.CreateNameContract(identity, name)
}

// GetNodeTwin returns the twin ID for a node ID
func (s *SubstrateDevImpl) GetNodeTwin(id uint32) (uint32, error) {
	node, err := s.Substrate.GetNode(id)
	if err != nil {
		return 0, normalizeNotFoundErrors(err)
	}
	return uint32(node.TwinID), nil
}

// UpdateNodeContract updates a node contract
func (s *SubstrateDevImpl) UpdateNodeContract(identity Identity, contract uint64, body string, hash string) (uint64, error) {
	res, err := s.Substrate.UpdateNodeContract(identity, contract, body, hash)
	return res, normalizeNotFoundErrors(err)
}

// CreateNodeContract creates a node contract
func (s *SubstrateDevImpl) CreateNodeContract(identity Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error) {
	res, err := s.Substrate.CreateNodeContract(identity, node, body, hash, publicIPs, solutionProviderID)
	return res, normalizeNotFoundErrors(err)
}

// GetContract returns a contract given its ID
func (s *SubstrateDevImpl) GetContract(contractID uint64) (Contract, error) {
	contract, err := s.Substrate.GetContract(contractID)
	return &DevContract{contract}, normalizeNotFoundErrors(err)
}

// CancelContract cancels a contract
func (s *SubstrateDevImpl) CancelContract(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return normalizeNotFoundErrors(err)
	}
	return nil
}

// EnsureContractCanceled ensures a canceled contract
func (s *SubstrateDevImpl) EnsureContractCanceled(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return normalizeNotFoundErrors(err)
	}
	return nil
}

// DeleteInvalidContracts deletes invalid contracts
func (s *SubstrateDevImpl) DeleteInvalidContracts(contracts map[uint32]uint64) error {
	for node, contractID := range contracts {
		valid, err := s.IsValidContract(contractID)
		// TODO: handle pause
		if err != nil {
			return normalizeNotFoundErrors(err)
		}
		if !valid {
			delete(contracts, node)
		}
	}
	return nil
}

// IsValidContract checks if a contract is invalid
func (s *SubstrateDevImpl) IsValidContract(contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = normalizeNotFoundErrors(err)
	// TODO: handle pause
	if errors.Is(err, ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}

// InvalidateNameContract invalidate a name contract
func (s *SubstrateDevImpl) InvalidateNameContract(
	ctx context.Context,
	identity Identity,
	contractID uint64,
	name string,
) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = normalizeNotFoundErrors(err)
	if errors.Is(err, ErrNotFound) {
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
			return 0, errors.Wrap(normalizeNotFoundErrors(err), "failed to cleanup unmatched name contract")
		}
		return 0, nil
	}

	return contractID, nil
}

// Close closes substrate
func (s *SubstrateDevImpl) Close() {
	s.Substrate.Close()
}
