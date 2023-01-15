package state

type DB interface {
	// Load should retrieve local state
	Load() error
	// GetState
	GetState() StateI
	// Save should save networks data to local state
	Save() error
	// Delete should delete networks state
	Delete() error
}

type StateI interface {
	// GetNetworks retrieves network state from local state
	GetNetworkState() NetworkState
}

type NetworkState interface {
	// GetNetwork retrieves network `networkName` from network state
	GetNetwork(networkName string) NetworkInterface
	// DeleteNetwork deletes `networkName` from local state
	DeleteNetwork(networkName string)
}

// NetworkInterface is an interface for network
type NetworkInterface interface {
	// GetNodeSubnet retrieves node's subnet from network local state
	GetNodeSubnet(nodeID uint32) string
	// SetNodeSubnet sets node's subnet in network local state
	SetNodeSubnet(nodeID uint32, subnet string)
	// DeleteNodeSubnet deletes node's subnet from network local state
	DeleteNodeSubnet(nodeID uint32)
	// GetNodeDeploymentHostIDs retrieves all node's used host id
	GetUsedNetworkHostIDs(nodeID uint32) []byte
	// GetDeploymentHostIDs retrieves deployment's used ips
	GetDeploymentHostIDs(nodeID uint32, deploymentID string) []byte
	// SetDeploymentHostIDs sets deployment's used ips
	SetDeploymentHostIDs(nodeID uint32, deploymentID string, ips []byte)
	// RemoveDeployment deletes deployment entry
	DeleteDeploymentHostIDs(nodeID uint32, deploymentID string)
}
