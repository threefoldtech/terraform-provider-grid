package integrationtests

import (
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestMattermostDeployment(t *testing.T) {
	/* Test case for deploying a matermost.

	   **Test Scenario**

	   - Deploy a matermost.
	   - Check that the outputs not empty.
	   - Check that vm is reachable.
	   - Make sure mattermost zinit service is running.
	   - Destroy the deployment.
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./mattermost",
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
	fqdn := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, fqdn)

	// Check that the solution is running successfully
	output, err := RemoteRun("root", planetary, "zinit list", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, output, "mattermost: Running")
	time.Sleep(60 * time.Second) // Sleeps for 60 seconds
	_, err = http.Get(fqdn)
	assert.NoError(t, err)

}
