// Package state provides a state to save the user work in a database.
package state

// State struct
type State struct {
	Networks NetworkMap `json:"networks"`
}

// GetNetworkState gets network state (names and their networks)
func (s *State) GetNetworkState() NetworkState {
	if s.Networks == nil {
		s.Networks = make(NetworkMap)
	}
	return s.Networks
}

// NewState generates a new state
func NewState() State {
	state := State{
		Networks: make(NetworkMap),
	}
	return state
}
