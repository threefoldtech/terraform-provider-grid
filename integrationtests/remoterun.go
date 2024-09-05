//go:build integration
// +build integration

// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"golang.org/x/crypto/ssh"
)

func setup() (deployer.TFPluginClient, error) {
	mnemonic := os.Getenv("MNEMONIC")

	network := os.Getenv("NETWORK")
	if network == "" {
		network = "dev"
	}

	return deployer.NewTFPluginClient(mnemonic, deployer.WithNetwork(network), deployer.WithLogs())
}

// RemoteRun used for running cmd remotely using ssh
func RemoteRun(user string, addr string, cmd string, privateKey string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", errors.Wrapf(err, "could not parse ssh private key %v", key)
	}
	// Authentication
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	// Connect
	port := "22"
	client, err := ssh.Dial("tcp", net.JoinHostPort(addr, port), config)
	if err != nil {
		return "", errors.Wrapf(err, "could not start ssh connection")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", errors.Wrapf(err, "could not create new session with message error")
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "could not execute command on remote with output %s", output)
	}
	return string(output), nil
}

// GenerateSSHKeyPair creats the public and private key for the machine
func GenerateSSHKeyPair() (string, string, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", errors.Wrapf(err, "could not generate rsa key")
	}

	pemKey := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}
	privateKey := pem.EncodeToMemory(pemKey)

	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "Couldn't extract public key")
	}
	authorizedKey := ssh.MarshalAuthorizedKey(pub)
	return string(authorizedKey), string(privateKey), nil
}
