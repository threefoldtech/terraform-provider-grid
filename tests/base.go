package tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/goombaio/namegenerator"
	"golang.org/x/crypto/ssh"
)

// UpWg used for up wireguard
func UpWg(wgConfig string) {
	f, err := os.Create("/tmp/test.conf")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(wgConfig)

	if err2 != nil {
		log.Fatal(err2)
	}

	cmd := exec.Command("sudo", "wg-quick", "up", "/tmp/test.conf")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}

// DownWG used for down wireguard
func DownWG() {
	cmd := exec.Command("sudo", "wg-quick", "down", "/tmp/test.conf")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(stdout))
}

// RemoteRun used for ssh host
func RemoteRun(user string, addr string, cmd string) (string, error) {
	privateKey := os.Getenv("PRIVATEKEY")
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", err
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
		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output
	err = session.Run(cmd)
	return b.String(), err
}

func VerifyIPs(wgConfig string, verifyIPs []string) bool {
	UpWg(wgConfig)

	for i := 0; i < len(verifyIPs); i++ {
		out, _ := exec.Command("ping", verifyIPs[i], "-c 5", "-i 3", "-w 10").Output()
		if strings.Contains(string(out), "Destination Host Unreachable") {
			return false
		}
	}

	for i := 0; i < len(verifyIPs); i++ {
		res, _ := RemoteRun("root", verifyIPs[i], "ifconfig")
		if !strings.Contains(string(res), verifyIPs[i]) {
			return false
		}
	}
	return true
}

func RandomName() string {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)

	name := nameGenerator.Generate()

	return name
}

func Wait(addr string, port string) bool {
	for t := time.Now(); time.Since(t) < 3*time.Minute; {
		_, err := net.DialTimeout("tcp", net.JoinHostPort(addr, "22"), time.Second*12)
		if err == nil {
			return true
		}
	}
	return true
}

func SshKeys() {
	os.Mkdir("/tmp/.ssh", 0755)
	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-f", "/tmp/.ssh/id_rsa", "-q")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(stdout))

	private_key, err := ioutil.ReadFile("/tmp/.ssh/id_rsa")
	if err != nil {
		log.Fatal(err)
	}

	public_key, e := ioutil.ReadFile("/tmp/.ssh/id_rsa.pub")
	if e != nil {
		log.Fatal(err)
	}

	os.Setenv("PUBLICKEY", string(public_key))
	os.Setenv("PRIVATEKEY", string(private_key))
}
