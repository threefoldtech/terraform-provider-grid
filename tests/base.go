package tests

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

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
