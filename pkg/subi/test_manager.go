package subi

import (
	"github.com/threefoldtech/substrate-client"
	subtest "github.com/threefoldtech/substrate-client-test"
)

// TestManager is test substrate manager
type TestManager struct {
	substrate.Manager
	versioned subtest.Manager
}

// NewTestManager generates a new test substrate manager
func NewTestManager(url ...string) Manager {
	return &TestManager{substrate.NewManager(url...), subtest.NewManager(url...)}
}

// SubstrateExt returns test substrate manager interface for executable functions
func (m *TestManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateTestImpl{sub}, err
}
