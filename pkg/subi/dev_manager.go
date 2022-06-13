package subi

import (
	"github.com/threefoldtech/substrate-client"
	subdev "github.com/threefoldtech/substrate-client-dev"
)

type Manager interface {
	substrate.Manager
	SubstrateExt() (SubstrateExt, error)
}

type DevManager struct {
	substrate.Manager
	versioned subdev.Manager
}

func NewDevManager(url ...string) Manager {
	return &DevManager{substrate.NewManager(url...), subdev.NewManager(url...)}
}

func (m *DevManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateDevImpl{sub}, err
}
