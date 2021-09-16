package test

import (
	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestZdbsDeployment(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a zdbs.
	   - Check that the outputs not empty.
	   - Destroy the deployment

	*/

	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	deploymentId := terraform.Output(t, terraformOptions, "deployment_id")
	assert.NotEmpty(t, deploymentId)

	zdb1Endpoint := terraform.Output(t, terraformOptions, "zdb1_endpoint")
	assert.NotEmpty(t, zdb1Endpoint)

	zdb1Namespace := terraform.Output(t, terraformOptions, "zdb1_namespace")
	assert.NotEmpty(t, node1Container2IP)
}
