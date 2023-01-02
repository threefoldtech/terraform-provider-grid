// Package state provides a state to save the user work in a database.
package state

import "encoding/json"

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

func (s *State) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// Unmarshal
func (s *State) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &s)
}

// NewState generates a new state
func NewState() State {
	state := State{
		Networks: make(NetworkMap),
	}
	return state
}
