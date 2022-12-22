package test

import (
	"log"
	"testing"

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
	pk, sk, err := tests.SshKeys()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": pk,
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

	err = tests.Wait(ygg_ip, "22")
	assert.NoError(t, err)

	verifyIPs := []string{ygg_ip, metrics}
	err = tests.VerifyIPs("", verifyIPs, sk)
	assert.NoError(t, err)

	res, err := tests.RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test && cat test", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "test")
}
