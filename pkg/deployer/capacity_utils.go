package deployer

import (
	"fmt"

	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// CapacityPrettyPrint prints the capacity data
func CapacityPrettyPrint(cap gridtypes.Capacity) string {
	return fmt.Sprintf("[mru: %d, sru: %d, hru: %d]", cap.MRU, cap.SRU, cap.HRU)
}

// AddCapacity adds a new data for capacity
func AddCapacity(cap *proxyTypes.Capacity, add *gridtypes.Capacity) {
	cap.CRU += add.CRU
	cap.MRU += add.MRU
	cap.SRU += add.SRU
	cap.HRU += add.HRU
}
