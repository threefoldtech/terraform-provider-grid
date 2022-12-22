package tests

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// UpWg used for setting up the wireguard interface
func UpWg(wgConfig string) (string, error) {
	f, err := os.Create("/tmp/test.conf")
	if err != nil {
		return "", errors.Wrapf(err, "error creating file")
	}
	defer f.Close()

	_, err = f.WriteString(wgConfig)
	if err != nil {
		return "", errors.Wrapf(err, "error writing wireguard config string")
	}

	cmd := exec.Command("sudo", "wg-quick", "up", "/tmp/test.conf")
	stdout, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "error excuting wg-quick up")
	}
	return string(stdout), nil
}

// DownWG used for tearing down the wireguard interface
func DownWG() (string, error) {
	cmd := exec.Command("sudo", "wg-quick", "down", "/tmp/test.conf")
	stdout, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "error excuting wg-quick down ")
	}
	return string(stdout), nil
}

// RemoteRun used for running cmd remotly using ssh
func RemoteRun(user string, addr string, cmd string, sk string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(sk))
	if err != nil {
		return "", errors.Wrapf(err, "error parsing ssh private key")
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
	client, err := ssh.Dial("tcp", net.JoinHostPort(addr, "22"), config)
	if err != nil {
		return "", errors.Wrapf(err, "error starting ssh connection ")
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", errors.Wrapf(err, "error creating new session")
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output
	err = session.Run(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "error excuting command on remote")
	}
	return b.String(), nil
}

// TODO: investigate if this is needed
func VerifyIPs(wgConfig string, verifyIPs []string, sk string) error {

	for i := 0; i < len(verifyIPs); i++ {
		out, err := exec.Command("ping", verifyIPs[i], "-c 5", "-i 3", "-w 10").Output()
		if strings.Contains(string(out), "Destination Host Unreachable") {
			return errors.Wrapf(err, "error host unreachable")
		}
	}
	for i := 0; i < len(verifyIPs); i++ {
		res, err := RemoteRun("root", verifyIPs[i], "ifconfig", sk)
		if err != nil {
			return errors.Wrapf(err, "error connecting to ip")
		}
		if !strings.Contains(string(res), verifyIPs[i]) {
			return errors.Wrapf(err, "error verifying  ips")
		}
	}
	return nil
}

// TODO: investigate if this is needed
// tries to connect to the provided address and port with time out
func Wait(addr string, port string) error {
	var err error
	for t := time.Now(); time.Since(t) < 3*time.Minute; {
		_, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
		if err == nil {
			return nil
		}
	}
	return errors.Wrapf(err, "couldn't join port")
}

// creating the public and private key for the machine
func SshKeys() (string, string, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", errors.Wrapf(err, "error generating rsa key")
	}

	// generate and write private key as PEM
	var privKeyBuf strings.Builder

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", errors.Wrapf(err, "error encoding private key")
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "error extracting public key")
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), privKeyBuf.String(), nil
}
