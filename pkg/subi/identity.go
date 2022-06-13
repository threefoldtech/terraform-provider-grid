package subi

import (
	subv2 "github.com/threefoldtech/substrate-client/v2"
)

type Identity subv2.Identity

func NewIdentityFromEd25519Phrase(phrase string) (Identity, error) {
	id, err := subv2.NewIdentityFromEd25519Phrase(phrase)
	return Identity(id), err
}

func NewIdentityFromSr25519Phrase(phrase string) (Identity, error) {
	id, err := subv2.NewIdentityFromSr25519Phrase(phrase)
	return Identity(id), err
}
