// Package state provides a state to save the user work in a database.
package state

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/pkg/errors"
)

// DBType number
type DBType int

const (
	// FileName is a static file name for state
	FileName = "state.json"
	// TypeFile is the type of db
	TypeFile DBType = iota
)

// FileDB struct is the state file for DB
type FileDB struct {
	st State
}

// NewLocalStateFileDB generates a new local state
func NewLocalStateFileDB() FileDB {
	return FileDB{}

}

// Load loads state from state.json file
func (f *FileDB) Load() error {
	// os.OpenFile(FileName, os.O_CREATE, 0644)
	f.st = State{}
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
