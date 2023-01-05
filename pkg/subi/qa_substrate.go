package subi

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	subqa "github.com/threefoldtech/substrate-client-qa"
)

// SubstrateQAImpl struct to use qa substrate
type SubstrateQAImpl struct {
	*subqa.Substrate
}

// GetContractIDByNameRegistration returns contract ID using its name
func (s *SubstrateQAImpl) GetContractIDByNameRegistration(name string) (uint64, error) {
	res, err := s.Substrate.GetContractIDByNameRegistration(name)
	return res, isNotFoundErrors(err)
}

// GetTwinIP returns twin IP given its ID
func (s *SubstrateQAImpl) GetTwinIP(id uint32) (string, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return "", isNotFoundErrors(err)
	}
	return twin.IP, nil
}

// GetTwinPK returns twin's public key
func (s *SubstrateQAImpl) GetTwinPK(id uint32) ([]byte, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return nil, isNotFoundErrors(err)
	}
	return twin.Account.PublicKey(), nil
}

// GetAccount returns the user's account
func (s *SubstrateQAImpl) GetAccount(identity Identity) (types.AccountInfo, error) {
	res, err := s.Substrate.GetAccount(identity)
	return res, isNotFoundErrors(err)
}

// CreateNameContract creates a new name contract
func (s *SubstrateQAImpl) CreateNameContract(identity Identity, name string) (uint64, error) {
	return s.Substrate.CreateNameContract(identity, name)
}

// GetNodeTwin returns the twin ID for a node ID
func (s *SubstrateQAImpl) GetNodeTwin(id uint32) (uint32, error) {
	node, err := s.Substrate.GetNode(id)
	if err != nil {
		return 0, isNotFoundErrors(err)
	}
	return uint32(node.TwinID), nil
}

// UpdateNodeContract updates a node contract
func (s *SubstrateQAImpl) UpdateNodeContract(identity Identity, contract uint64, body string, hash string) (uint64, error) {
	res, err := s.Substrate.UpdateNodeContract(identity, contract, body, hash)
	return res, isNotFoundErrors(err)
}

// CreateNodeContract creates a node contract
func (s *SubstrateQAImpl) CreateNodeContract(identity Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error) {
	res, err := s.Substrate.CreateNodeContract(identity, node, body, hash, publicIPs, solutionProviderID)
	return res, isNotFoundErrors(err)
}

// GetContract returns a contract given its ID
func (s *SubstrateQAImpl) GetContract(contractID uint64) (Contract, error) {
	contract, err := s.Substrate.GetContract(contractID)
	return &QAContract{contract}, isNotFoundErrors(err)
}

// CancelContract cancels a contract
func (s *SubstrateQAImpl) CancelContract(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return isNotFoundErrors(err)
	}
	return nil
}

// EnsureContractCanceled ensures a canceled contract
func (s *SubstrateQAImpl) EnsureContractCanceled(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return isNotFoundErrors(err)
	}
	return nil
}

// DeleteInvalidContracts deletes invalid contracts
func (s *SubstrateQAImpl) DeleteInvalidContracts(contracts map[uint32]uint64) error {
	for node, contractID := range contracts {
		valid, err := s.IsValidContract(contractID)
		// TODO: handle pause
		if err != nil {
			return isNotFoundErrors(err)
		}
		if !valid {
			delete(contracts, node)
		}
	}
	return nil
}

// IsValidContract checks if a contract is invalid
func (s *SubstrateQAImpl) IsValidContract(contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = isNotFoundErrors(err)
	// TODO: handle pause
	if errors.Is(err, ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}

// InvalidateNameContract invalidate a name contract
func (s *SubstrateQAImpl) InvalidateNameContract(
	ctx context.Context,
	identity Identity,
	contractID uint64,
	name string,
) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = isNotFoundErrors(err)
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
			return 0, errors.Wrap(isNotFoundErrors(err), "failed to cleanup unmatched name contract")
		}
		return 0, nil
	}

	return contractID, nil
}
