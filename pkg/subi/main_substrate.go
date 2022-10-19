package subi

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	submain "github.com/threefoldtech/substrate-client-main"
)

type SubstrateMainImpl struct {
	*submain.Substrate
}

func (s *SubstrateMainImpl) GetContractIDByNameRegistration(name string) (uint64, error) {
	res, err := s.Substrate.GetContractIDByNameRegistration(name)
	return res, terr(err)
}
func (s *SubstrateMainImpl) GetTwinIP(id uint32) (string, error) {
	twin, err := s.Substrate.GetTwin(id)
	if err != nil {
		return "", terr(err)
	}
	return twin.IP, nil
}
func (s *SubstrateMainImpl) GetAccount(identity Identity) (types.AccountInfo, error) {
	res, err := s.Substrate.GetAccount(identity)
	return res, terr(err)
}
func (s *SubstrateMainImpl) CreateNameContract(identity Identity, name string) (uint64, error) {
	return s.Substrate.CreateNameContract(identity, name)
}
func (s *SubstrateMainImpl) GetNodeTwin(id uint32) (uint32, error) {
	node, err := s.Substrate.GetNode(id)
	if err != nil {
		return 0, terr(err)
	}
	return uint32(node.TwinID), nil
}
func (s *SubstrateMainImpl) UpdateNodeContract(identity Identity, contract uint64, body string, hash string) (uint64, error) {
	res, err := s.Substrate.UpdateNodeContract(identity, contract, body, hash)
	return res, terr(err)
}
func (s *SubstrateMainImpl) CreateNodeContract(identity Identity, node uint32, body string, hash string, publicIPs uint32, solutionProviderID *uint64) (uint64, error) {
	res, err := s.Substrate.CreateNodeContract(identity, node, body, hash, publicIPs, solutionProviderID)
	return res, terr(err)
}
func (s *SubstrateMainImpl) GetContract(contractID uint64) (Contract, error) {
	contract, err := s.Substrate.GetContract(contractID)
	return &MainContract{contract}, terr(err)
}
func (s *SubstrateMainImpl) CancelContract(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return terr(err)
	}
	return nil
}
func (s *SubstrateMainImpl) EnsureContractCanceled(identity Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := s.Substrate.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return terr(err)
	}
	return nil
}

func (s *SubstrateMainImpl) DeleteInvalidContracts(contracts map[uint32]uint64) error {
	for node, contractID := range contracts {
		valid, err := s.IsValidContract(contractID)
		// TODO: handle pause
		if err != nil {
			return terr(err)
		}
		if !valid {
			delete(contracts, node)
		}
	}
	return nil
}

func (s *SubstrateMainImpl) IsValidContract(contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = terr(err)
	// TODO: handle pause
	if errors.Is(err, ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}

func (s *SubstrateMainImpl) InvalidateNameContract(
	ctx context.Context,
	identity Identity,
	contractID uint64,
	name string,
) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}
	contract, err := s.Substrate.GetContract(contractID)
	err = terr(err)
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
			return 0, errors.Wrap(terr(err), "failed to cleanup unmatching name contract")
		}
		return 0, nil
	}

	return contractID, nil
}
