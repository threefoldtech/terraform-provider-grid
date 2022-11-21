package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	mock "github.com/threefoldtech/terraform-provider-grid/internal/provider/mocks"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

func TestProvider(t *testing.T) {
	st := state.NewState()
	ctrl := gomock.NewController(t)
	subext := mock.NewMockSubstrateExt(ctrl)
	if err := New("dev", subext, &st)().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
