package integrationtests

import (
	"log"
	"testing"
)

func TestNetwork(t *testing.T) {
	_, _, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

}
