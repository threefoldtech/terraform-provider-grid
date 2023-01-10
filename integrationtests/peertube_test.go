//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestPeerTubeDeployment(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a peertube.
	   - Check that the outputs not empty.
	   - Check that vm is reachable
	   - Check that env variables set successfully
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Log(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./peertube",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	planetary := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, planetary)

	peertube := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, fqdn)

	err = tests.TestConnection(planetary, "22")
	assert.NoError(t, err)

	// Check that env variables set successfully
	output, err := tests.RemoteRun("root", planetary, "zinit list", sk)
	assert.NoError(t, err)
	assert.Contains(t, output, "peertube: Running")

}
