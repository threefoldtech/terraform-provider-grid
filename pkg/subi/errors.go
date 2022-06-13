package subi

import (
	"github.com/pkg/errors"
	subv2 "github.com/threefoldtech/substrate-client/v2"
	subv3 "github.com/threefoldtech/substrate-client/v3"
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
