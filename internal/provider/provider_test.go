// Package provider is the terraform provider
package provider

import (
	"testing"

	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

func TestProvider(t *testing.T) {
	stateDB := state.NewLocalFileState()
	f, sub := New("dev", &stateDB)
	if sub != nil {
		defer sub.Close()
	}
	if err := f().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
