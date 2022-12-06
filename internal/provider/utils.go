package provider

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
)

func isIn(l []uint16, i uint16) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInByte(l []byte, i byte) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInUint32(l []uint32, i uint32) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}
func isInUint64(l []uint64, i uint64) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func isInStr(l []string, i string) bool {
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
