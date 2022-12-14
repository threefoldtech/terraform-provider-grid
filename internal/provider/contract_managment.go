package provider

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func IsValidNameContract(sub subi.Substrate, contractID uint64) (bool, error) {
	contract, err := sub.GetContract(contractID)
	if errors.Is(err, substrate.ErrNotFound) {
		return false, nil
	}
	if err == nil && !contract.State.IsCreated {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "couldn't get contract %d info", contractID)
	}
	return true, nil
}

func IsValidCapacityReservationContract(sub subi.Substrate, contractID uint64) error {
	contract, err := sub.GetContract(contractID)
	if err != nil {
		return errors.Wrapf(err, "couldn't get capacity reservation contract %d info", contractID)
	}
	if !contract.State.IsCreated {
		return fmt.Errorf("capacity reservation contract %d not in a created state", contractID)
	}
	return nil
}

func IsValidDeployment(sub subi.Substrate, deploymentID uint64) (bool, error) {
	_, err := sub.GetDeployment(deploymentID)

	if err != nil && err != substrate.ErrNotFound {
		return false, errors.Wrapf(err, "couldn't get deployment %d info", deploymentID)
	}
	if err == substrate.ErrNotFound {
		return false, nil
	}
	return true, nil
}

func IsInvalidContract(sub *substrate.Substrate, contractID uint64, deploymentID uint64) (err error) {
	err = IsValidCapacityReservationContract(sub, contractID)
	if err != nil {
		return err
	}
	valid, err := IsValidDeployment(sub, deploymentID)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("deployment with id %d is not valid", deploymentID)
	}
	return nil
}
func CheckInvalidContracts(sub *substrate.Substrate, deployments map[uint64]uint64) (err error) {
	for contractID, deploymentID := range deployments {
		err := IsInvalidContract(sub, contractID, deploymentID)
		if err != nil {
			return err
		}
	}
	return nil
}

func InvalidateNameContract(sub subi.Substrate, identity substrate.Identity, contractID uint64, name string) (uint64, error) {
	if contractID == 0 {
		return 0, nil
	}

	valid, err := IsValidNameContract(sub, contractID)
	if err != nil {
		return 0, errors.Wrapf(err, "Couldn't validate name contract for contract ", contractID)
	}
	if !valid {
		return 0, errors.Wrapf(err, "Name contract isn't valid for contract ", contractID)
	}

	contract, err := sub.GetContract(contractID)

	if errors.Is(err, substrate.ErrNotFound) {
		return 0, errors.Wrapf(err, "couldn't find error for contract ", contractID)
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

func EnsureContractCanceled(sub subi.Substrate, identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := sub.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return err
	}
	return nil
}
