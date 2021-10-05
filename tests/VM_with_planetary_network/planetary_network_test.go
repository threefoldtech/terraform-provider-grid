package test

import (
	"os"
	"testing"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSingleNodeDeployment(t *testing.T) {
	/* Test case for deployeng a VM with planetary network.

	   **Test Scenario**

	   - Deploy a VM in single node.
	   - Check that the outputs not empty.
	   - Verify the VMs ips
	   - Check that ygg ip is reachable.
	   - ssh to VM and check if yggdrasil is active
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
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
	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1Container1IP)

	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	verifyIPs := []string{publicIP, node1Container1IP, yggIP}
	tests.VerifyIPs(wgConfig, verifyIPs)
	defer tests.DownWG()

	// ssh to VM and check if yggdrasil is active
	res, _ := tests.RemoteRun("root", yggIP, "systemctl status yggdrasil")
	assert.Contains(t, string(res), "active (running)")
}
