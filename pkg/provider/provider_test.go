package provider

import (
	"testing"

	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

func TestProvider(t *testing.T) {
	st := state.NewState()
	f, sub := New("dev", &st)
	if sub != nil {
		defer sub.Close()
	}
	if err := f().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
