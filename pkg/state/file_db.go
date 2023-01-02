// Package state provides a state to save the user work in a database.

package state

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/pkg/errors"
)

type DBType int

const (
	// FILE_NAME is a static file name for state
	FILE_NAME        = "state.json"
	TypeFile  DBType = iota
)

// ErrWrongDBType is an error for wrong db type
var ErrWrongDBType = errors.New("wrong db type")

// NewLocalStateDB generates a new local state
func NewLocalStateDB(t DBType) (*fileDB, error) {
	if t == TypeFile {
		return &fileDB{}, nil
	}
	return nil, ErrWrongDBType
}

// Load loads state from state.json file
func (f *fileDB) Load() error {
	// os.OpenFile(FILE_NAME, os.O_CREATE, 0644)
	f.st = &State{}
	_, err := os.Stat(FILE_NAME)
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
func (f *FileDB) GetState() State {
	if reflect.DeepEqual(f.st, State{}) {
		state := NewState()
		f.st = state
	}
	return f.st
}

// Save saves the state to the state,json file
func (f *FileDB) Save() error {
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
func (f *FileDB) Delete() error {
	return os.Remove(FileName)
}
