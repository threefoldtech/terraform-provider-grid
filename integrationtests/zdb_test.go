//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestZdbsDeployment(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a zdbs.
	   - Deploy a VM (have a IPv6)
	   - Check that the outputs not empty.
	   - Check that zdb reachable from VM.
	   - Destroy the deployment

	*/

	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./zdbs",
		Parallelism:  1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	deploymentId := terraform.Output(t, terraformOptions, "deployment_id")
	assert.NotEmpty(t, deploymentId)

	zdb1Endpoint := terraform.Output(t, terraformOptions, "zdb1_endpoint")
	assert.NotEmpty(t, zdb1Endpoint)

	zdb1Namespace := terraform.Output(t, terraformOptions, "zdb1_namespace")
	assert.NotEmpty(t, zdb1Namespace)

	// Check that zdb reachable from VM.

}
