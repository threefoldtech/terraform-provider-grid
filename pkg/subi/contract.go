package subi

import (
	subdev "github.com/threefoldtech/substrate-client-dev"
	submain "github.com/threefoldtech/substrate-client-main"
	subqa "github.com/threefoldtech/substrate-client-qa"
	subtest "github.com/threefoldtech/substrate-client-test"
)

// Contract is a contract interface
type Contract interface {
	IsDeleted() bool
	IsCreated() bool
	TwinID() uint32
	PublicIPCount() uint32
}

// DevContract is for dev contract
type DevContract struct {
	*subdev.Contract
}

// IsDeleted checks if contract is deleted
func (c *DevContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}

// IsCreated checks if contract is created
func (c *DevContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

// TwinID returns contract's twin ID
func (c *DevContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

// PublicIPCount returns contract's public IPs count
func (c *DevContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

// QAContract is for qa contract
type QAContract struct {
	*subqa.Contract
}

// IsDeleted checks if contract is deleted
func (c *QAContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}

// IsCreated checks if contract is created
func (c *QAContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

// TwinID returns contract's twin ID
func (c *QAContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

// PublicIPCount returns contract's public IPs count
func (c *QAContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

// TestContract is for test net contract
type TestContract struct {
	*subtest.Contract
}

// IsDeleted checks if contract is deleted
func (c *TestContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}

// IsCreated checks if contract is created
func (c *TestContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

// TwinID returns contract's twin ID
func (c *TestContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

// PublicIPCount returns contract's public IPs count
func (c *TestContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}

// MainContract is for main net contract
type MainContract struct {
	*submain.Contract
}

// IsDeleted checks if contract is deleted
func (c *MainContract) IsDeleted() bool {
	return c.Contract.State.IsDeleted
}

// IsCreated checks if contract is created
func (c *MainContract) IsCreated() bool {
	return c.Contract.State.IsCreated
}

// TwinID returns contract's twin ID
func (c *MainContract) TwinID() uint32 {
	return uint32(c.Contract.TwinID)
}

// PublicIPCount returns contract's public IPs count
func (c *MainContract) PublicIPCount() uint32 {
	return uint32(c.Contract.ContractType.NodeContract.PublicIPsCount)
}
