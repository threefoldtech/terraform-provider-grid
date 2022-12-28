//go:build integration
// +build integration

package test

import (
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
	"os"
	"os/exec"
	"testing"
)

func TestMattermostDeployment(t *testing.T) {
	/* Test case for deploying a matermost.

	   **Test Scenario**

	   - Deploy a matermost.
	   - Check that the outputs not empty.
	   - Check that vm is reachable
	   - Check that env variables set successfully
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SSHKeys()
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
	ip := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, ip)

	status := false
	status = tests.Wait(ip, "22")
	if status == false {
		t.Errorf("public ip not reachable")
	}

	out, _ := exec.Command("ping", ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", ip, "cat /proc/1/environ")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	// Check that the solution is running successfully

	res1, _ := tests.RemoteRun("root", ip, "zinit list")
	assert.Contains(t, res1, "mattermost: Running")

}
