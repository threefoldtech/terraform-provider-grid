// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// GatewayFQDNProxy for gateway FQDN proxy
type GatewayFQDNProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string

	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool

	// Backends are list of backend ips
	Backends []zos.Backend

	// FQDN deployed on the node
	FQDN string
}

// GatewayFQDNProxyFromZosWorkload generates a gateway FQDN proxy from a zos workload
func GatewayFQDNProxyFromZosWorkload(wl gridtypes.Workload) (GatewayFQDNProxy, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return GatewayFQDNProxy{}, errors.Wrap(err, "failed to get workload data")
	}
	data := dataI.(*zos.GatewayFQDNProxy)

	return GatewayFQDNProxy{
		Name:           wl.Name.String(),
		TLSPassthrough: data.TLSPassthrough,
		Backends:       data.Backends,
		FQDN:           data.FQDN,
	}, nil
}

// ZosWorkload generates a zos workload from GatewayFQDNProxy
func (g *GatewayFQDNProxy) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Type:    zos.GatewayFQDNProxyType,
		Name:    gridtypes.Name(g.Name),
		// REVISE: whether description should be set here
		Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			TLSPassthrough: g.TLSPassthrough,
			Backends:       g.Backends,
			FQDN:           g.FQDN,
		}),
	}
}
