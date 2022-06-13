package subi

import (
	"github.com/threefoldtech/substrate-client"
	subv2 "github.com/threefoldtech/substrate-client/v2"
)

type Manager interface {
	substrate.Manager
	SubstrateExt() (SubstrateExt, error)
}

type ManagerV2 struct {
	substrate.Manager
	versioned subv2.Manager
}

func NewManagerV2(url ...string) Manager {
	return &ManagerV2{substrate.NewManager(url...), subv2.NewManager(url...)}
}

func (m *ManagerV2) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateImplV2{sub}, err
}
