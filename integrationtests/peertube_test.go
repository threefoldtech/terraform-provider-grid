//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/integrationtests"
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
		log.Fatal(err)
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
	ip := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, ip)

	err = tests.Wait(ip, "22")
	assert.NoError(t, err)

	out, _ := exec.Command("ping", ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// Check that env variables set successfully
	res, err := tests.RemoteRun("root", ip, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	res, err = tests.RemoteRun("root", ip, "zinit list", sk)
	assert.NoError(t, err)
	assert.Contains(t, res, "peertube: Running")

}
