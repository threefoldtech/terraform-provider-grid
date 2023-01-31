//go:build integration
// +build integration

// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// UpWg used for setting up the wireguard interface
func UpWg(wgConfig, wgConfDir string) (string, error) {

	f, err := os.Create(path.Join(wgConfDir, "test.conf"))
	if err != nil {
		return "", errors.Wrapf(err, "could not create file")
	}

	_, err = f.WriteString(wgConfig)
	if err != nil {
		return "", errors.Wrapf(err, "could not write on file")
	}

	_, err = exec.Command("wg-quick", "up", f.Name()).Output()
	if err != nil {
		return "", errors.Wrapf(err, "could not excute wg-quick up with"+f.Name())
	}

	return f.Name(), nil

}

// DownWG used for tearing down the wireguard interface
func DownWG(wgConfDir string) (string, error) {
	cmd := exec.Command("wg-quick", "down", path.Join(wgConfDir, "test.conf"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "could not excute wg-quick down with output %s", out)
	}
	return string(out), nil
}

// AreWGIPsReachable used to check if wire guard ip is reachable
func AreWGIPsReachable(wgConfig string, ipsToCheck []string, privateKey string) error {
	g := new(errgroup.Group)
	for _, ip := range ipsToCheck {
		ip := ip
		g.Go(func() error {
			output, err := RemoteRun("root", ip, "ifconfig", privateKey)
			if err != nil {
				return errors.Wrapf(err, "could not connect as a root user to the machine with ip %s with output %s", ip, output)
			}
			if !strings.Contains(string(output), ip) {
				return errors.Wrapf(err, "ip %s could not be verified. ifconfig output: %s", ip, output)
			}
			return nil
		})
	}
	return g.Wait()
}
