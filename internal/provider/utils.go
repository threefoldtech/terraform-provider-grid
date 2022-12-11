package provider

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

type Number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

type Chars interface {
	byte | string
}

func includes[N Number | Chars](l []N, i N) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func IsValidContract(sub *substrate.Substrate, contractID uint64) (bool, error) {
	if contractID == 0 {
		return false, nil
	}
	contract, err := sub.GetContract(contractID)
	if errors.Is(err, substrate.ErrNotFound) || (contract != nil && !contract.State.IsCreated) {
		return false, nil
	} else if err != nil {
		return true, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}
func DeleteInvalidContracts(sub *substrate.Substrate, contracts map[uint32]uint64) (err error) {
	for nodeId, contractID := range contracts {
		valid, err := IsValidContract(sub, contractID)
		if err != nil {
			return err
		}
		if !valid {
			delete(contracts, nodeId)
		}

	}
	return nil
}

func InvalidateNameContract(sub *substrate.Substrate, identity substrate.Identity, contractID uint64, name string) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}
	contract, err := sub.GetContract(contractID)
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
		err := sub.CancelContract(identity, contractID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to cleanup unmatching name contract")
		}
		return 0, nil
	}

	return contractID, nil
}

func EnsureContractCanceled(sub *substrate.Substrate, identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := sub.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return err
	}
	return nil
}

func WrapErrors(err1, err2 error) error {
	if err1 != nil && err2 == nil {
		return err1
	} else if err1 == nil && err2 != nil {
		return err2
	} else if err1 != nil && err2 != nil {
		return fmt.Errorf("%w. %w", err1, err2)
	}
	return nil
}
