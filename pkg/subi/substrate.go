package subi

import (
	"math/big"
	"net"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/threefoldtech/substrate-client"
)

type SubstrateClient interface {
	EnsureAccount(identity substrate.Identity, activationURL string, termsAndConditionsLink string, terminsAndConditionsHash string) (info types.AccountInfo, err error)
	GetAccount(identity substrate.Identity) (info types.AccountInfo, err error)
	GetCurrentHeight() (uint32, error)
	FetchEventsForBlockRange(start uint32, end uint32) (types.StorageKey, []types.StorageChangeSet, error)
	GetBlock(block types.Hash) (*types.SignedBlock, error)
	ProposeBurnTransactionOrAddSig(identity substrate.Identity, txID uint64, target string, amount *big.Int, signature string, stellarAddress string, sequence_number uint64) (*types.Call, error)
	SetBurnTransactionExecuted(identity substrate.Identity, txID uint64) (*types.Call, error)
	GetBurnTransaction(identity substrate.Identity, burnTransactionID types.U64) (*substrate.BurnTransaction, error)
	IsBurnedAlready(identity substrate.Identity, burnTransactionID types.U64) (exists bool, err error)
	CreateNodeContract(identity substrate.Identity, node uint32, body []byte, hash string, publicIPs uint32) (uint64, error)
	CreateNameContract(identity substrate.Identity, name string) (uint64, error)
	UpdateNodeContract(identity substrate.Identity, contract uint64, body []byte, hash string) (uint64, error)
	CancelContract(identity substrate.Identity, contract uint64) error
	GetContract(id uint64) (*substrate.Contract, error)
	GetContractWithHash(node uint32, hash string) (uint64, error)
	GetContractIDByNameRegistration(name string) (uint64, error)
	GetNodeContracts(node uint32) ([]types.U64, error)
	GetDepositFee(identity substrate.Identity) (int64, error)
	GetEntity(id uint32) (*substrate.Entity, error)
	GetFarm(id uint32) (*substrate.Farm, error)
	GetClient() (substrate.Conn, substrate.Meta, error)
	IsMintedAlready(identity substrate.Identity, mintTxID string) (exists bool, err error)
	ProposeOrVoteMintTransaction(identity substrate.Identity, txID string, target substrate.AccountID, amount *big.Int) (*types.Call, error)
	GetNodeByTwinID(twin uint32) (uint32, error)
	GetNode(id uint32) (*substrate.Node, error)
	CreateNode(identity substrate.Identity, node substrate.Node) (uint32, error)
	UpdateNode(identity substrate.Identity, node substrate.Node) (uint32, error)
	CreateRefundTransactionOrAddSig(identity substrate.Identity, tx_hash string, target string, amount int64, signature string, stellarAddress string, sequence_number uint64) (*types.Call, error)
	SetRefundTransactionExecuted(identity substrate.Identity, txHash string) (*types.Call, error)
	IsRefundedAlready(identity substrate.Identity, txHash string) (exists bool, err error)
	GetRefundTransaction(identity substrate.Identity, txHash string) (*substrate.RefundTransaction, error)
	GetTwinByPubKey(pk []byte) (uint32, error)
	GetTwin(id uint32) (*substrate.Twin, error)
	CreateTwin(identity substrate.Identity, ip net.IP) (uint32, error)
	UpdateTwin(identity substrate.Identity, ip net.IP) (uint32, error)
	GetUser(id uint32) (*substrate.User, error)
	Call(cl substrate.Conn, meta substrate.Meta, identity substrate.Identity, call types.Call) (hash types.Hash, err error)
}
