package test

import (
	"testing"

	"os"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a QSFS.
	   **Test Scenario**
	   - Deploy a qsfs.
	   - Check that the outputs not empty.
	   - Check that containers is reachable
	   - Check that env variables set successfully
	   - get the qsfs one and find its path and do some operations there you should can writing/reading/listing
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

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", ygg_ip, "printenv")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	res, err := tests.RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test && cat test")
	assert.Empty(t, err)
	assert.Contains(t, string(res), "test")
}
