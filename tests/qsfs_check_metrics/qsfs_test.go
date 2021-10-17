package test

import (
	"os/exec"
	"testing"

	"os"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
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

	ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, ygg_ip)

	verifyIPs := []string{ygg_ip, metrics}
	tests.VerifyIPs("", verifyIPs)

	// get metrics
	cmd := exec.Command("curl " + metrics)
	output, _ := cmd.CombinedOutput()

	// try write to a file in mounted disk
	_, err := tests.RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test")
	assert.Empty(t, err)

	// get metrics after write
	cmd2 := exec.Command("curl " + metrics)
	output_after_write, _ := cmd2.CombinedOutput()

	// check that syscalls for write should increase
	assert.NotEqual(t, output, output_after_write)
}
