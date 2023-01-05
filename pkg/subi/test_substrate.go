package subi

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	subtest "github.com/threefoldtech/substrate-client-test"
)

// SubstrateTestImpl struct to use test substrate
type SubstrateTestImpl struct {
	*subtest.Substrate
}

// GetContractIDByNameRegistration returns contract ID using its name
func (s *SubstrateTestImpl) GetContractIDByNameRegistration(name string) (uint64, error) {
	res, err := s.Substrate.GetContractIDByNameRegistration(name)
	return res, isNotFoundErrors(err)
}

// GetTwinIP returns twin IP given its ID
func (s *SubstrateTestImpl) GetTwinIP(id uint32) (string, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return "", isNotFoundErrors(err)
	}
	return twin.IP, nil
}

// GetTwinPK returns twin's public key
func (s *SubstrateTestImpl) GetTwinPK(id uint32) ([]byte, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return nil, isNotFoundErrors(err)
	}
	return twin.Account.PublicKey(), nil
}

// GetAccount returns the user's account
func (s *SubstrateTestImpl) GetAccount(identity Identity) (types.AccountInfo, error) {
	res, err := s.Substrate.GetAccount(identity)
	return res, isNotFoundErrors(err)
}

// CreateNameContract creates a new name contract
func (s *SubstrateTestImpl) CreateNameContract(identity Identity, name string) (uint64, error) {
	return s.Substrate.CreateNameContract(identity, name)
}

// GetNodeTwin returns the twin ID for a node ID
func (s *SubstrateTestImpl) GetNodeTwin(id uint32) (uint32, error) {
	node, err := s.Substrate.GetNode(id)
	if err != nil {
		return 0, isNotFoundErrors(err)
	}
	return uint32(node.TwinID), nil
}

// UpdateNodeContract updates a node contract
func (s *SubstrateTestImpl) UpdateNodeContract(identity Identity, contract uint64, body string, hash string) (uint64, error) {
	res, err := s.Substrate.UpdateNodeContract(identity, contract, body, hash)
	return res, isNotFoundErrors(err)
}

// CreateNodeContract creates a node contract
func (s *SubstrateTestImpl) CreateNodeContract(identity Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error) {
	res, err := s.Substrate.CreateNodeContract(identity, node, body, hash, publicIPs, solutionProviderID)
	return res, isNotFoundErrors(err)
}

// GetContract returns a contract given its ID
func (s *SubstrateTestImpl) GetContract(contractID uint64) (Contract, error) {
	contract, err := s.Substrate.GetContract(contractID)
	return &TestContract{contract}, isNotFoundErrors(err)
}

// CancelContract cancels a contract
func (s *SubstrateTestImpl) CancelContract(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return isNotFoundErrors(err)
	}
	return nil
}

// EnsureContractCanceled ensures a canceled contract
func (s *SubstrateTestImpl) EnsureContractCanceled(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return isNotFoundErrors(err)
	}
	return nil
}

// DeleteInvalidContracts deletes invalid contracts
func (s *SubstrateTestImpl) DeleteInvalidContracts(contracts map[uint32]uint64) error {
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
func (s *SubstrateTestImpl) IsValidContract(contractID uint64) (bool, error) {
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
func (s *SubstrateTestImpl) InvalidateNameContract(
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
