package subi

import (
	"github.com/threefoldtech/substrate-client"
	submain "github.com/threefoldtech/substrate-client-main"
)

type MainManager struct {
	substrate.Manager
	versioned submain.Manager
}

func NewMMainanager(url ...string) Manager {
	return &MainManager{substrate.NewManager(url...), submain.NewManager(url...)}
}

func (m *MainManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateMainImpl{sub}, err
}
