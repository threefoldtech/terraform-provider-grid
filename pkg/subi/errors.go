package subi

import (
	"github.com/pkg/errors"
	subv2 "github.com/threefoldtech/substrate-client-dev"
	subv3 "github.com/threefoldtech/substrate-client-test"
)

var ErrNotFound = subv2.ErrNotFound
var ErrAccountNotFound = subv2.ErrAccountNotFound

func terr(err error) error {
	if errors.Is(err, subv2.ErrNotFound) || errors.Is(err, subv3.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, subv2.ErrAccountNotFound) || errors.Is(err, subv3.ErrAccountNotFound) {
		return ErrAccountNotFound
	}
	return err
}
