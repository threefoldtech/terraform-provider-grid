package test

import (
	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestPreSearchDeployment(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a presearch.
	   - Check that the outputs not empty.
	   - Check that node is reachable.
	   - Check that vm is reachable
	   - Check that env variables set successfully
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
	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)
	publicIP, err := tests.IPFromCidr(publicIP)
	assert.NoError(t, err)
	// Check that vm is reachable
	err = tests.Wait(publicIP, "22")
	assert.NoError(t, err)

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", publicIP, "cat /proc/1/environ")
	assert.Contains(t, string(res), "PRESEARCH_REGISTRATION_CODE=e5083a8d0a6362c6cf7a3078bfac81e3")

	time.Sleep(60 * time.Second) // Sleeps for 60 seconds

	res1, _ := tests.RemoteRun("root", publicIP, "zinit list")
	assert.Contains(t, res1, "prenode: Success")
}
