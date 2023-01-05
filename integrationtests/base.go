package integrationtests

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
	"testing"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// UpWg used for setting up the wireguard interface
func UpWg(wgConfig, wgConfDir string, t *testing.T) (string, error) {
	tempdir := t.TempDir()
	f, err := os.Create(tempdir + "/test.conf")
	if err != nil {
		return "", errors.Wrapf(err, "error creating file")
	}
	_, err = f.WriteString(wgConfig)
	if err != nil {
		return "", errors.Wrapf(err, "error writing file")
	}
	cmd := exec.Command("wg-quick", "up", f.Name())
	_, err = cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "error executing wg-quick up")
	}

	return f.Name(), nil

}

// DownWG used for tearing down the wireguard interface
func DownWG(conf string) (string, error) { //tempdir
	cmd := exec.Command("wg-quick", "down", conf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "error executing wg-quick down")
	}
	return string(out), nil
}

// RemoteRun used for running cmd remotly using ssh
func RemoteRun(user string, addr string, cmd string, privateKey string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", errors.Wrapf(err, "error parsing ssh private key %w", key)
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
		return "", errors.Wrapf(err, "error creating new session")
	}
	defer session.Close()
	var b bytes.Buffer
	session.Stdout = &b
	// trying to make compound output work insted of buffer

	err = session.Run(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "error executing command on remote")
	}
	return b.String(), nil
}

// TODO: investigate if this is needed
func IsIPReachable(wgConfig string, isIPReachable []string, privateKey string) error {
	for i := range isIPReachable {
		out, err := exec.Command("ping", isIPReachable[i], "-c 5", "-i 3", "-w 10").Output()
		if err != nil {
			return errors.Wrapf(err, "error executing command on remote")
		}
		if strings.Contains(string(out), "Destination Host Unreachable") {
			return errors.Wrapf(err, "error host unreachable")
		}
	}
	for i := 0; i < len(isIPReachable); i++ {
		res, err := RemoteRun("root", isIPReachable[i], "ifconfig", privateKey)
		if err != nil {
			return errors.Wrapf(err, "couldn't connect as a root user to the machine")
		}
		if !strings.Contains(string(res), isIPReachable[i]) {
			return errors.Wrapf(err, "the ip is not reachable and couldnt be verified ")
		}
	}
	return nil
}

// creating the public and private key for the machine
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
func TestConnection(addr string, port string) error {
	var err error
	for t := time.Now(); time.Since(t) < 3*time.Minute; {
		con, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
		con.Close()
		if err == nil {
			return nil
		}
	}
	return err
}
