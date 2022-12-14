package provider

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
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

func getNodeIdByCapacityId(sub subi.Substrate, capacityId uint64) (uint32, error) {
	contract, err := sub.GetContract(capacityId)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to getNodeIdByCapacity, capacityId: (%d)", capacityId)
	}
	nodeId := uint32(contract.ContractType.CapacityReservationContract.NodeID)
	return nodeId, nil
}
