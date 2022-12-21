package test

import (
	"os"
	"strings"
	"testing"
	"time"

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
	masterIP, err := tests.IPFromCidr(masterPublicIP)
	assert.NoError(t, err, "error parsing master node ip")
	err = tests.Wait(masterIP, "22")
	assert.NoError(t, err, "can not reach master node")
	// ssh to master node
	time.Sleep(30 * (time.Second))
	res, err := tests.RemoteRun("root", masterIP, "kubectl get node")
	res = strings.Trim(res, "\n")
	assert.Empty(t, err, "can not ssh to master node, IP: ", masterIP)

	// Check worker deployed number
	nodes := strings.Split(string(res), "\n")[1:]
	assert.Equal(t, 2, len(nodes)) // assert that there are 1 worker and 1 master

	// Check that worker is ready
	for i := 0; i < len(nodes); i++ {
		assert.Contains(t, nodes[i], "Ready")
	}
}
