package subi

import (
	"github.com/threefoldtech/substrate-client"
)

type Manager interface {
	substrate.Manager
	SubstrateExt() (SubstrateExt, error)
}

type manager struct {
	substrate.Manager
}

func NewManager(url ...string) Manager {
	return &manager{substrate.NewManager(url...)}
}

func (m *manager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.Substrate()
	return &SubstrateImpl{sub}, err
}
