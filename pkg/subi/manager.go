// subi package exposes substrate functionality
package subi

import (
	"github.com/threefoldtech/substrate-client"
)

// Manager interface to expose SubstrateExt
type Manager interface {
	substrate.Manager
	SubstrateExt() (SubstrateExt, error)
}

type manager struct {
	substrate.Manager
}

// Create NewManager
func NewManager(url ...string) Manager {
	return &manager{substrate.NewManager(url...)}
}

func (m *manager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.Substrate()
	return &SubstrateImpl{sub}, err
}
