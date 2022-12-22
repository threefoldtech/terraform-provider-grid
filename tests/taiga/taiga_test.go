package test

import (
	"log"
	"os/exec"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
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
	publicIp := terraform.Output(t, terraformOptions, "node1_zmachine1_ygg_ip")
	assert.NotEmpty(t, publicIp)

	webIp := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, webIp)

	// Check that vm is reachable
	publicIp = strings.Split(publicIp, "/")[0]
	err = tests.Wait(publicIp, "22")
	assert.NoError(t, err)

	// Check that env variables set successfully
	res, err := tests.RemoteRun("root", publicIp, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	res, err = tests.RemoteRun("root", publicIp, "zinit list", sk)
	assert.NoError(t, err)
	assert.Contains(t, res, "taiga: Running")

	//check the webpage
	err = tests.Wait(webIp, "22")
	assert.NoError(t, err)

	out1, _ := exec.Command("ping", webIp, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")
}
