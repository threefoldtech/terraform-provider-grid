package integrationtests

import (
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// UpWg used for setting up the wireguard interface
func UpWg(wgConfig, wgConfDir string) (string, error) {

	f, err := os.Create(path.Join(wgConfDir, "test.conf"))
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
		return "", errors.Wrapf(err, "error executing wg-quick up with"+f.Name())
	}

	return f.Name(), nil

}

// DownWG used for tearing down the wireguard interface
func DownWG(wgConfDir string) (string, error) { //tempdir
	cmd := exec.Command("wg-quick", "down", path.Join(wgConfDir, "test.conf"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "error executing wg-quick down with output %s", out)
	}
	return string(out), nil
}

// AreWGIPsReachable used to check if wire guard ip is reachable
func AreWGIPsReachable(wgConfig string, ipsToCheck []string, privateKey string) error {
	errChannel := make(chan error, len(ipsToCheck))
	var wg sync.WaitGroup
	for _, ip := range ipsToCheck {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			output, err := RemoteRun("root", ip, "ifconfig", privateKey)
			if err != nil {
				errChannel <- errors.Wrapf(err, "couldn't connect as a root user to the machine with ip %s with output %s", ip, output)
				return
			}
			if !strings.Contains(string(output), ip) {
				errChannel <- errors.Wrapf(err, "ip %s couldnt be verified. ifconfig output: %s", ip, output)
				return
			}
		}(ip)
	}
	wg.Wait()
	return <-errChannel
}
