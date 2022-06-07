package deployer

import (
	"context"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

// EnsureNameContract checks and if not creates a name contract
// returns an error if this name is owned by another contract of another twin
func EnsureNameContract(
	ctx context.Context,
	sub subi.SubstrateClient,
	identity substrate.Identity,
	twinID uint32,
	name string,
) (uint64, error) {
	contractID, err := sub.GetContractIDByNameRegistration(name)
	if errors.Is(err, substrate.ErrNotFound) {
		contractID, err := sub.CreateNameContract(identity, name)
		return contractID, errors.Wrap(err, "failed to create name contract")
	} else if err != nil {
		return 0, errors.Wrapf(err, "couldn't get the owning contract id of the name %s", name)
	}
	contract, err := sub.GetContract(contractID)
	if err != nil {
		return 0, errors.Wrapf(err, "couldn't get the owning contract of the name %s", name)
	}
	if contract.TwinID != types.U32(twinID) {
		return 0, errors.Wrapf(err, "name already registered by twin id %d with contract id %d", contract.TwinID, contractID)
	}
	return contractID, nil
}

func InvalidateNameContract(
	ctx context.Context,
	sub subi.SubstrateClient,
	identity substrate.Identity,
	contractID uint64,
	name string,
) (uint64, error) {
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
