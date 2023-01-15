package subi

import (
	"github.com/threefoldtech/substrate-client"
	subdev "github.com/threefoldtech/substrate-client-dev"
)

// Manager is substrate manager
type Manager interface {
	substrate.Manager
	SubstrateExt() (SubstrateExt, error)
}

// DevManager is dev substrate manager
type DevManager struct {
	substrate.Manager
	versioned subdev.Manager
}

// NewDevManager generates a new dev substrate manager
func NewDevManager(url ...string) Manager {
	return &DevManager{substrate.NewManager(url...), subdev.NewManager(url...)}
}

// SubstrateExt returns dev substrate manager interface for executable functions
func (m *DevManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateDevImpl{sub}, err
}
