// Package state provides a state to save the user work in a database.
package state

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/state"
)

// Getter interface for local state
type Getter interface {
	// GetState
	GetState() *state.NetworkState
}

const (
	// FileName is a static file name for state that is generated beside the .tf file
	FileName = "state.json"
)

// LocalFileState struct is the local state file
type LocalFileState struct {
	st *state.NetworkState
}

// NewLocalFileState generates a new local state
func NewLocalFileState() LocalFileState {
	return LocalFileState{}

}

// Load loads state from state.json file
func (f *LocalFileState) Load(FileName string) error {
	// os.OpenFile(FileName, os.O_CREATE, 0644)
	f.st = &state.NetworkState{}
	_, err := os.Stat(FileName)
	if err != nil && os.IsNotExist(err) {
		_, err = os.OpenFile(FileName, os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		return nil
	}
	content, err := os.ReadFile(FileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, &f.st)
	if err != nil {
		return err
	}
	return nil
}

// GetState returns the current state
func (f *LocalFileState) GetState() *state.NetworkState {
	if reflect.DeepEqual(f.st, &state.NetworkState{}) {
		state := &state.NetworkState{
			State: make(map[string]state.Network),
		}
		f.st = state
	}
	return f.st
}

// Save saves the state to the state,json file
func (f *LocalFileState) Save(FileName string) error {
	if content, err := json.Marshal(f.st); err == nil {
		err = os.WriteFile(FileName, content, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write file: %s", FileName)
		}
	} else {
		return errors.Wrapf(err, "failed to save file: %s", FileName)
	}
	return nil
}

// Delete deletes state,json file
func (f *LocalFileState) Delete(FileName string) error {
	return os.Remove(FileName)
}
