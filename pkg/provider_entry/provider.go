package providerentry

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func New(version string, st state.StateI) (func() *schema.Provider, subi.SubstrateExt) {
	return provider.New(version, st)
}
