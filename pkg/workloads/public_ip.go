// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ConstructPublicIPWorkload constructs a public IP workload
func ConstructPublicIPWorkload(workloadName string, ipv4 bool, ipv6 bool) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(workloadName),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: ipv4,
			V6: ipv6,
		}),
	}
}
