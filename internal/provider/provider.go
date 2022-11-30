package provider

import (
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type ApiClient struct {
	Twin_id       uint32
	Mnemonics     string
	Substrate_url string
	Rmb_redis_url string
	Use_rmb_proxy bool
	Grid_client   proxy.Client
	Rmb           rmb.Client
	SubstrateConn *substrate.Substrate
	Manager       substrate.Manager
	Identity      substrate.Identity
	State         state.StateI
}
