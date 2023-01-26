//go:build integration
// +build integration

package integrationtests

import (
	"net/http"
	"testing"
	"time"

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

	statusOk := false
	ticker := time.NewTicker(2 * time.Second)
	for now := time.Now(); time.Since(now) < 1*time.Minute && !statusOk; {
		<-ticker.C
		resp, err := http.Get(peertube)
		if err == nil && resp.StatusCode == 200 {
			statusOk = true
		}
	}
	assert.True(t, statusOk, "website did not respond with 200 status code")
}
