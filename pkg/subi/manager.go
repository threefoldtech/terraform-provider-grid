package subi

import "github.com/threefoldtech/substrate-client"

type Manager struct {
	substrate.Manager
}

func NewManager(url ...string) Manager {
	return Manager{substrate.NewManager(url...)}
}

func (m *Manager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.Manager.Substrate()
	return &SubstrateImpl{sub}, err
}
