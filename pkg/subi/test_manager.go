package subi

import (
	"github.com/threefoldtech/substrate-client"
	subtest "github.com/threefoldtech/substrate-client-test"
)

type TestManager struct {
	substrate.Manager
	versioned subtest.Manager
}

func NewTestManager(url ...string) Manager {
	return &TestManager{substrate.NewManager(url...), subtest.NewManager(url...)}
}

func (m *TestManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateTestImpl{sub}, err
}
