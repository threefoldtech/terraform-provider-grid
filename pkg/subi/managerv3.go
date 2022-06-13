package subi

import (
	"github.com/threefoldtech/substrate-client"
	subv3 "github.com/threefoldtech/substrate-client/v3"
)

type ManagerV3 struct {
	substrate.Manager
	versioned subv3.Manager
}

func NewManagerV3(url ...string) Manager {
	return &ManagerV3{substrate.NewManager(url...), subv3.NewManager(url...)}
}

func (m *ManagerV3) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateImplV3{sub}, err
}
