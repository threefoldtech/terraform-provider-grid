package integrationtests

import (
	"log"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestQSFS(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("qsfs", func(t *testing.T) {
		/* Test case for deployeng a QSFS check metrics.
		   **Test Scenario**
		   - Deploy a qsfs.
		   - Check that the outputs not empty.
		   - Check that containers is reachable
		   - write to a file. number of syscalls for write should increase try open, read, create, rename or etc number of syscalls should increase
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs_check_metrics",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		assert.NotEmpty(t, metrics)

		ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, ygg_ip)

		// get metrics
		cmd := exec.Command("curl", metrics)
		output, _ := cmd.Output()

		// try write to a file in mounted disk
		_, err = RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test", privateKey)
		assert.NoError(t, err)
		// get metrics after write
		cmd = exec.Command("curl", metrics)
		output, _ = cmd.Output()

		// check that syscalls for write should increase
		assert.NotEqual(t, output, output)
	})

	t.Run("qsfs_read_write", func(t *testing.T) {
		/* Test case for deployeng a QSFS.
		**Test Scenario**
		- Deploy a qsfs.
		- Check that the outputs not empty.
		- Check that containers is reachable
		- get the qsfs one and find its path and do some operations there you should can writing/reading/listing
		- Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs_read_write",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		assert.NotEmpty(t, metrics)

		ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, ygg_ip)

		output, err := RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test && cat test", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "test")
	})

}
