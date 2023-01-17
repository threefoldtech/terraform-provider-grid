package integrationtests

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestMount(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("Mounts", func(t *testing.T) {
		/* Test case for deployeng a disk and mount it.

		   **Test Scenario**

		   - Deploy a disk.
		   - Check that the outputs not empty.
		   - Verify the VMs ips.
		   - Check that disk has been mounted successfully with 10G.
		   - Destroy the deployment.

		*/

		// retryable errors in terraform testing.
		// generate ssh keys for test
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

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// Check that disk has been mounted successfully with 10G
		output, err = RemoteRun("root", planetary, "df -h | grep -w /app", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "10G")
	})

	t.Run("MountsWithBiggerFile", func(t *testing.T) {
		/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

		   **Test Scenario**

		   - Deploy a mount disk with size 1G.
		   - Check that the outputs not empty.
		   - ssh to VM and try to create a file with size 1G.
		   - Destroy the deployment
		*/
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

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// ssh to VM and try to create a file with size 1G.
		_, err = RemoteRun("root", planetary, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", privateKey)
		if err == nil {
			t.Errorf("should fail with out of memory")
		}
	})
}
