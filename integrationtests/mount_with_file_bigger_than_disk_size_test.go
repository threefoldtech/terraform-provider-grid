//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestMountWithBiggerFileDeployment(t *testing.T) {
	/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

	   **Test Scenario**

	   - Deploy a mount disk with size 1G.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - ssh to VM and try to create a file with size 1G.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	publicKey, privateKey, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./mount_with_file_bigger_than_disk_size",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApplyE(t, terraformOptions)

	planetary := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, planetary)

	//test connection to the vm
	err = tests.TestConnection(planetary, "22")
	assert.NoError(t, err)

	// Check that env variables set successfully
	output, err := tests.RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "SSH_KEY")

	// ssh to VM and try to create a file with size 1G.
	output, err = tests.RemoteRun("root", planetary, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", privateKey)
	if err == nil {
		t.Errorf("should fail with out of memory")
	}
}
