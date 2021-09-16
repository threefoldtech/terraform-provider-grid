package test

import (
	"os"
	"testing"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSingleMountDeployment(t *testing.T) {
	/* Test case for deployeng a disk and mount it.

	   **Test Scenario**

	   - Deploy a disk.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that containers is reachable.
	   - Verify the VMs ips.
	   - Check that env variables set successfully
	   - Check that disk has been mounted successfully with 10G.
	   - Destroy the deployment.

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

	node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1Container2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	// Check that containers is reachable
	verifyIPs := []string{publicIP, node1Container2IP, node1Container1IP}
	tests.VerifyIPs(wgConfig, verifyIPs)
	defer tests.DownWG()

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", node1Container1IP, "printenv")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	// Check that disk has been mounted successfully with 10G
	res1, errors3 := tests.RemoteRun("root", node1Container2IP, "df -h | grep -w /app")
	assert.Empty(t, errors3)
	assert.Contains(t, string(res1), "10.0G")
}
