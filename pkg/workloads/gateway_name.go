// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// GatewayNameProxy struct for gateway name proxy
type GatewayNameProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string

	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool

	// Backends are list of backend ips
	Backends []zos.Backend

	// FQDN deployed on the node
	FQDN string
}

// GatewayNameProxyFromZosWorkload generates a gateway name proxy from a zos workload
func GatewayNameProxyFromZosWorkload(wl gridtypes.Workload) (GatewayNameProxy, error) {
	var result zos.GatewayProxyResult

	if len(wl.Result.Data) > 0 {
		if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
			return GatewayNameProxy{}, errors.Wrap(err, "error unmarshalling json")
		}
	}

	dataI, err := wl.WorkloadData()
	if err != nil {
		return GatewayNameProxy{}, errors.Wrap(err, "failed to get workload data")
	}
	data := dataI.(*zos.GatewayNameProxy)

	return GatewayNameProxy{
		Name:           data.Name,
		TLSPassthrough: data.TLSPassthrough,
		Backends:       data.Backends,
		FQDN:           result.FQDN,
	}, nil
}

// ZosWorkload generates a zos workload from GatewayNameProxy
func (g *GatewayNameProxy) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Type:    zos.GatewayNameProxyType,
		Name:    gridtypes.Name(g.Name),
		// REVISE: whether description should be set here
		Data: gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name:           g.Name,
			TLSPassthrough: g.TLSPassthrough,
			Backends:       g.Backends,
		}),
	}
}
