package deployer

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func EnsureContractCanceled(sub subi.SubstrateClient, identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := sub.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return err
	}
	return nil
}

func DeleteInvalidContracts(sub subi.SubstrateClient, contracts map[uint32]uint64) error {
	for node, contractID := range contracts {
		valid, err := IsValidContract(sub, contractID)
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

func IsValidContract(sub subi.SubstrateClient, contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := sub.GetContract(contractID)
	// TODO: handle pause
	if errors.Is(err, substrate.ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contract)
	}
	return true, nil
}
