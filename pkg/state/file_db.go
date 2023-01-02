// Package state provides a state to save the user work in a database.

package state

import (
	"os"

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

type fileDB struct {
	st StateI
}

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
		_, err = os.OpenFile(FILE_NAME, os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		return nil
	}
	content, err := os.ReadFile(FILE_NAME)
	if err != nil {
		return err
	}
	err = f.st.Unmarshal(content)
	if err != nil {
		return err
	}
	return nil
}

// GetState returns the current state
func (f *fileDB) GetState() StateI {
	if f.st == nil {
		state := NewState()
		f.st = &state
	}
	return f.st
}

// Save saves the state to the state,json file
func (f *fileDB) Save() error {
	if content, err := f.st.Marshal(); err == nil {
		err = os.WriteFile(FILE_NAME, content, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write file: %s", FILE_NAME)
		}
	} else {
		return errors.Wrapf(err, "failed to save file: %s", FILE_NAME)
	}
	return nil
}

// Delete deletes state,json file
func (f *fileDB) Delete() error {
	return os.Remove(FILE_NAME)
}
