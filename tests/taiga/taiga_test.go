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

func TestTaigaDeployment(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a taiga.
	   - Check that the outputs not empty.
	   - Check that vm is reachable.
	   - Check that env variables set successfully.
	   - Ping the website.
	   - Destroy the deployment.
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
	publicIp := terraform.Output(t, terraformOptions, "node1_zmachine1_ygg_ip")
	assert.NotEmpty(t, publicIp)

	webIp := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, webIp)

	// Check that vm is reachable
	ip := publicIp
	status := false
	status = tests.Wait(ip, "22")
	if status == false {
		t.Errorf("public ip not reachable")
	}


	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", ip, "cat /proc/1/environ")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	res1, _ := tests.RemoteRun("root", ip, "zinit list")
	assert.Contains(t, res1, "taiga: Running")

	//check the webpage
	webip := webIp
	status1 := false
	status1 = tests.Wait(webip, "22")
	if status1 == false {
		t.Errorf("public ip not reachable")
	}
	out1, _ := exec.Command("ping", webip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")
}
