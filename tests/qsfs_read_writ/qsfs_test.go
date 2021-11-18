package test

import (
	"testing"

	"os"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a QSFS.
	   **Test Scenario**
	   - Deploy a qsfs.
	   - Check that the outputs not empty.
	   - Check that containers is reachable
	   - get the qsfs one and find its path and do some operations there you should can writing/reading/listing
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

	ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, ygg_ip)

	status := tests.Wait(ygg_ip, "22")
	for status == false {
		status = tests.Wait(ygg_ip, "22")
	}

	verifyIPs := []string{ygg_ip, metrics}
	tests.VerifyIPs("", verifyIPs)

	res, err := tests.RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test && cat test")
	assert.Empty(t, err)
	assert.Contains(t, string(res), "test")
}
