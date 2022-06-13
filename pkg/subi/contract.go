package subi

import (
	subv2 "github.com/threefoldtech/substrate-client/v2"
	subv3 "github.com/threefoldtech/substrate-client/v3"
)

type Contract interface {
	IsDeleted() bool
	IsCreated() bool
	TwinID() uint32
	PublicIPCount() uint32
}

type ContractV2 struct {
	*subv2.Contract
}

func (c *ContractV2) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *ContractV2) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *ContractV2) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *ContractV2) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

type ContractV3 struct {
	*subv3.Contract
}

func (c *ContractV3) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *ContractV3) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *ContractV3) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *ContractV3) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}
