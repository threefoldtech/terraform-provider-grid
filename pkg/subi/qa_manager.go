package subi

import (
	"github.com/threefoldtech/substrate-client"
	subqa "github.com/threefoldtech/substrate-client-qa"
)

type QAManager struct {
	substrate.Manager
	versioned subqa.Manager
}

func NewQAManager(url ...string) Manager {
	return &QAManager{substrate.NewManager(url...), subqa.NewManager(url...)}
}

func (m *QAManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateQAImpl{sub}, err
}
