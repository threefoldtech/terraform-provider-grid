package state

// DB interface for database
type DB interface {
	// Load should retrieve local state
	Load() error
	// GetState
	GetState() State
	// Save should save networks data to local state
	Save() error
	// Delete should delete networks state
	Delete() error
}

// StateI interface for state
type StateI interface {
	// GetNetworks retrieves network state from local state
	GetNetworkState() NetworkState
}
