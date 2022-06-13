package subi

import (
	subdev "github.com/threefoldtech/substrate-client-dev"
	submain "github.com/threefoldtech/substrate-client-main"
	subqa "github.com/threefoldtech/substrate-client-qa"
	subtest "github.com/threefoldtech/substrate-client-test"
)

type Contract interface {
	IsDeleted() bool
	IsCreated() bool
	TwinID() uint32
	PublicIPCount() uint32
}

type DevContract struct {
	*subdev.Contract
}

func (c *DevContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *DevContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *DevContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *DevContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

type QAContract struct {
	*subqa.Contract
}

func (c *QAContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *QAContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *QAContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *QAContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

type TestContract struct {
	*subtest.Contract
}

func (c *TestContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *TestContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *TestContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *TestContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

type MainContract struct {
	*submain.Contract
}

func (c *MainContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}
func (c *MainContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

func (c *MainContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

func (c *MainContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}
