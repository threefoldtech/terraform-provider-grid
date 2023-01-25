//go:build integration
// +build integration

package integrationtests

import (
	"net/http"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestPeerTubeDeployment(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a peertube.
	   - Check that the outputs not empty.
	   - Check that vm is reachable
	   - Check that peertube service is running
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./peertube",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	planetary := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, planetary)

	peertube := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, peertube)

	ok := TestConnection(planetary, "22")
	assert.True(t, ok)

	// Check that env variables set successfully
	output, err := RemoteRun("root", planetary, "zinit list", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, output, "peertube: Running")
	_, err = http.Get(peertube)
	assert.NoError(t, err)

}
