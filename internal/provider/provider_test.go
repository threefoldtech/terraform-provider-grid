package provider

import (
	"testing"

	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

func TestProvider(t *testing.T) {
	st := state.NewState()
	if err := New("dev", &st)().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
