// Package provider is the terraform provider
package provider

import (
	"fmt"
	"log"
	"math/big"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

// validateAccount checks the mnemonics is associated with an account with key type ed25519
func validateAccount(apiClient *apiClient, sub subi.SubstrateExt) error {
	_, err := sub.GetAccount(apiClient.identity)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}
	if err != nil { // Account not found
		funcs := map[string]func(string) (substrate.Identity, error){"ed25519": substrate.NewIdentityFromEd25519Phrase, "sr25519": substrate.NewIdentityFromSr25519Phrase}
		for keyType, f := range funcs {
			ident, err2 := f(apiClient.mnemonics)
			if err2 != nil { // shouldn't happen, return original error
				log.Printf("couldn't convert the mnemomincs to %s key: %s", keyType, err2.Error())
				return err
			}
			_, err2 = sub.GetAccount(ident)
			if err2 == nil { // found an identity with key type other than the provided
				return fmt.Errorf("found an account with %s key type and the same mnemonics, make sure you provided the correct key type", keyType)
			}
		}
		// didn't find an account with any key type
		return err
	}
	return nil
}

func validateRMBProxyServer(apiClient *apiClient) error {
	return apiClient.grid_client.Ping()
}

func validateRMBProxy(apiClient *apiClient) error {
	if err := validateRMBProxyServer(apiClient); err != nil {
		return err
	}
	return nil
}

func preValidate(apiClient *apiClient, sub subi.SubstrateExt) error {
	return validateRMBProxy(apiClient)
}

func validateAccountMoneyForExtrinsics(sub subi.SubstrateExt, identity substrate.Identity) error {
	acc, err := sub.GetAccount(identity)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}
	log.Printf("money %d\n", acc.Data.Free)
	if acc.Data.Free.Cmp(big.NewInt(20000)) == -1 {
		return fmt.Errorf("account contains %s, min fee is 20000", acc.Data.Free)
	}
	return nil
}
