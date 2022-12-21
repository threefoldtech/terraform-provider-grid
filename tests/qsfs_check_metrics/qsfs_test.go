package test

import (
	"os/exec"
	"testing"

	"os"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a QSFS check metrics.
	   **Test Scenario**
	   - Deploy a qsfs.
	   - Check that the outputs not empty.
	   - Check that containers is reachable
	   - write to a file. number of syscalls for write should increase try open, read, create, rename or etc number of syscalls should increase
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
	metrics := terraform.Output(t, terraformOptions, "metrics")
	assert.NotEmpty(t, metrics)

	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	verifyIPs := []string{yggIP}
	err := tests.VerifyIPs("", verifyIPs)
	assert.NoError(t, err, "ips not reachable")
	// get metrics
	cmd := exec.Command("curl", metrics)
	output, _ := cmd.Output()

	// try write to a file in mounted disk
	_, err = tests.RemoteRun("root", yggIP, "cd /qsfs && echo test >> test")
	assert.Empty(t, err)

	// get metrics after write
	cmd2 := exec.Command("curl", metrics)
	output_after_write, _ := cmd2.Output()

	// check that syscalls for write should increase
	assert.NotEqual(t, output, output_after_write)
}
