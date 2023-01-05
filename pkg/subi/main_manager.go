package subi

import (
	"github.com/threefoldtech/substrate-client"
	submain "github.com/threefoldtech/substrate-client-main"
)

// MainManager is main substrate manager
type MainManager struct {
	substrate.Manager
	versioned submain.Manager
}

// NewMainManager generates a new main substrate manager
func NewMainManager(url ...string) Manager {
	return &MainManager{substrate.NewManager(url...), submain.NewManager(url...)}
}

// SubstrateExt returns main substrate manager interface for executable functions
func (m *MainManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateMainImpl{sub}, err
}
