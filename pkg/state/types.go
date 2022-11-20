package state

import "errors"

type DBType int

const (
	TypeFile DBType = iota
)

var ErrWrongDBType = errors.New("wrong db type")

type DB interface {
	// LoadState should retrieve local state
	Load() error
	// GetState
	GetState() StateI
	// Save should save networks data to local state
	Save() error
	// Delete should delete networks state
	Delete() error
}

// implementors (file, dbms, ...)

type StateI interface {
	// GetNetworks retrieves network state from local state
	GetNetworkState() NetworkState
	// Marshal
	Marshal() ([]byte, error)
	// Unmarshal
	Unmarshal(data []byte) error
}

type NetworkState interface {
	// GetNetwork retrieves network `networkName` from network state
	GetNetwork(networkName string) Network
	// DeleteNetwork deletes `networkName` from local state
	DeleteNetwork(networkName string)
}

type Network interface {
	// GetNodeSubnet retrieves node's subnet from network local state
	GetNodeSubnet(nodeID uint32) string
	// SetNodeSubnet sets node's subnet in network local state
	SetNodeSubnet(nodeID uint32, subnet string)
	// DeleteNodeSubnet deletes node's subnet from network local state
	DeleteNodeSubnet(nodeID uint32)
	// GetNodeIPs retrieves all node's used ips
	GetNodeIPsList(nodeID uint32) []byte
	// GetDeploymentIPs retrieves deployment's used ips
	GetDeploymentIPs(nodeID uint32, deploymentID string) []byte
	// SetDeploymentIPs sets deployment's used ips
	SetDeploymentIPs(nodeID uint32, deploymentID string, ips []byte)
	// RemoveDeployment deletes deployment entry
	DeleteDeployment(nodeID uint32, deploymentID string)
	//
	GetSubnets() map[uint32]string
	GetNodeIPs() NodeIPs
}

func NewLocalStateDB(t DBType) (DB, error) {
	if t == TypeFile {
		return &fileDB{}, nil
	}
	// TODO
	return nil, ErrWrongDBType
}
