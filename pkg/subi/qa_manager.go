package subi

import (
	"github.com/threefoldtech/substrate-client"
	subqa "github.com/threefoldtech/substrate-client-qa"
)

// QAManager is qa substrate manager
type QAManager struct {
	substrate.Manager
	versioned subqa.Manager
}

// NewQAManager generates a new qa substrate manager
func NewQAManager(url ...string) Manager {
	return &QAManager{substrate.NewManager(url...), subqa.NewManager(url...)}
}

// SubstrateExt returns qa substrate manager interface for executable functions
func (m *QAManager) SubstrateExt() (SubstrateExt, error) {
	sub, err := m.versioned.Substrate()
	return &SubstrateQAImpl{sub}, err
}
