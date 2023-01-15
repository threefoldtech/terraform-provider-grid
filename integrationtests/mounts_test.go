//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
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
	// generate ssh keys for test
	publicKey, privateKey, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./mounts",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	planetary := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, planetary)

	node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1Container1IP)

	node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1Container2IP)

	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	//testing connections
	ok := tests.TestConnection(planetary, "22")
	assert.True(t, ok)

	// Check that env variables set successfully
	output, err := tests.RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "SSH_KEY")

	// Check that disk has been mounted successfully with 10G
	output, err = tests.RemoteRun("root", planetary, "df -h | grep -w /app", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "10G")
}
