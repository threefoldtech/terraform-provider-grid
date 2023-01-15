package integrationtests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// RemoteRun used for running cmd remotly using ssh
func RemoteRun(user string, addr string, cmd string, privateKey string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", errors.Wrapf(err, "error parsing ssh private key %v", key)
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
		return "", errors.Wrapf(err, "error starting ssh connection ")
	}
	session, err := client.NewSession()
	if err != nil {
		return "", errors.Wrapf(err, "error creating new session with message error ")
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "error executing command on remote with output %s", output)
	}
	return string(output), nil
}

// GenerateSSHKeyPair creats the public and private key for the machine
func GenerateSSHKeyPair() (string, string, error) {

	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", errors.Wrapf(err, "error generating rsa key")
	}

	pemKey := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}
	privateKey := pem.EncodeToMemory(pemKey)

	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "error extracting public key")
	}
	authorizedKey := ssh.MarshalAuthorizedKey(pub)
	return string(authorizedKey), string(privateKey), nil
}
