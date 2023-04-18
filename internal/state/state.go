// Package state provides a state to save the user work in a database.
package state

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"

// State struct
type State struct {
	Networks deployer.NetworkState `json:"networks"`
}

// GetNetworkState gets network state (names and their networks)
func (s *State) GetNetworkState() deployer.NetworkState {
	if s.Networks == nil {
		s.Networks = make(deployer.NetworkState)
	}
	return s.Networks
}

// NewState generates a new state
func NewState() State {
	return State{
		Networks: make(deployer.NetworkState),
	}
}
