package integrationtests

import (
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestQSFS(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("qsfs_test", func(t *testing.T) {
		/* Test case for deployeng a QSFS check metrics.
		   **Test Scenario**
		   - Deploy a qsfs.
		   - Check that the outputs not empty.
		   - Assert that qsfs create syscalls from metrics endpoint are equal to 0
		   - Write a file on qsfs
		   - Assert that qsfs create syscalls from metrics endpoint are equal to 1
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		assert.NotEmpty(t, metrics)

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		// get metrics
		cmd := exec.Command("curl", metrics)
		output, err := cmd.Output()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 0")

		// try write to a file in mounted disk
		_, err = RemoteRun("root", yggIP, "cd /qsfs && echo hamadatext >> hamadafile", privateKey)
		assert.NoError(t, err)

		// get metrics after write
		cmd = exec.Command("curl", metrics)
		output, err = cmd.Output()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 1")

	})
}
