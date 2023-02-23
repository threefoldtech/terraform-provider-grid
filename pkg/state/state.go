// Package state provides a state to save the user work in a database.
package state

// State struct
type State struct {
	Networks NetworkState `json:"networks"`
}

// GetNetworkState gets network state (names and their networks)
func (s *State) GetNetworkState() NetworkState {
	if s.Networks == nil {
		s.Networks = make(NetworkState)
	}
	return s.Networks
}

// NewState generates a new state
func NewState() State {
	return State{
		Networks: make(NetworkState),
	}
}
