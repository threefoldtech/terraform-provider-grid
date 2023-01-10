//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
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
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./presearch",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	publicIp := terraform.Output(t, terraformOptions, "computed_public_ip")
	assert.NotEmpty(t, publicIp)

	planetary := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, planetary)
	// Check that vm is reachable
	ip := strings.Split(publicIp, "/")[0]
	// err = tests.TestConnection(planetary, "22")
	// assert.NoError(t, err)

	// Check that env variables set successfully
	output, err := tests.RemoteRun("root", ip, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "PRESEARCH_REGISTRATION_CODE=e5083a8d0a6362c6cf7a3078bfac81e3")

	time.Sleep(60 * time.Second) // Sleeps for 60 seconds

	output, err = tests.RemoteRun("root", ip, "zinit list", sk)
	assert.NoError(t, err)
	assert.Contains(t, output, "prenode: Success")
}