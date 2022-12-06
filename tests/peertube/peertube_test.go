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

func TestKubernetesDeployment(t *testing.T) {
	/* Test case for deployeng a presearch.

	   **Test Scenario**

	   - Deploy a peertube.
	   - Check that the outputs not empty.
	   - Check that node is reachable.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SshKeys()
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

	// Check that vm is reachable
	// ip := strings.Split(publicIp, "/")[0]
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

	res1, _ := tests.RemoteRun("root", ip, "zinit list")
	assert.Contains(t, res1, "prepare-redis: Success")

}
