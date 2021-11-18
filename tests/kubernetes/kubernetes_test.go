package test

import (
	"os"
	"os/exec"
	"testing"

	"strings"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestKubernetesDeployment(t *testing.T) {
	/* Test case for deployeng a k8s.

	   **Test Scenario**

	   - Deploy a k8s.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that master is reachable
	   - Check workers deployed number.
	   - Check that workers is ready.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SshKeys()
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	masterPublicIP := terraform.Output(t, terraformOptions, "master_public_ip")
	assert.NotEmpty(t, masterPublicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// Check that master is reachable
	masterIP := strings.Split(masterPublicIP, "/")[0]
	status := false
	for i := 0; i < 5; i++ {
		status = tests.Wait(masterIP, "22")
		if status {
			break
		}
	}
	if status == false {
		t.Errorf("public ip not reachable")
	}

	out, _ := exec.Command("ping", masterPublicIP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to master node
	res, errors := tests.RemoteRun("root", masterIP, "kubectl get node")
	assert.Empty(t, errors)

	// Check worker deployed number
	nodes := strings.Split(string(res), "\n")
	workers := nodes[1:]               // remove header
	assert.Equal(t, len(workers)-1, 2) // assert that there are 1 worker and 1 master

	// Check that worker is ready
	for i := 0; i < len(workers)-1; i++ {
		assert.Contains(t, workers[i], "Ready")
	}
}
