//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSingleVMWithPlanetary(t *testing.T) {
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
	// generate ssh keys for test
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./planetary_network_test",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	// ssh to VM by ygg_ip
	output, err := RemoteRun("root", yggIP, "cat /proc/1/environ", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "TEST_VAR=this value for test")
}
