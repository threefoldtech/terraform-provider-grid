package state

import "encoding/json"

type state struct {
	Networks networkingState `json:"networks"`
}

func (s *state) GetNetworkState() NetworkState {
	if s.Networks == nil {
		s.Networks = make(networkingState)
	}
	return &s.Networks
}

func (s *state) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// Unmarshal
func (s *state) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &s)
}

func NewState() state {
	state := state{
		Networks: make(networkingState),
	}
	return state
}
