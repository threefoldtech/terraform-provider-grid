package test

import (
	"log"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestSingleNodeDeployment(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a singlenode.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that containers is reachable
	   - Verify the VMs ips
	   - Check that env variables set successfully.
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
	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1Container1IP)

	node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1Container2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	pIP := strings.Split(publicIP, "/")[0]
	err = tests.Wait(pIP, "22")
	assert.NoError(t, err)
	verifyIPs := []string{publicIP, node1Container1IP, node1Container1IP}
	_, err = tests.UpWg(wgConfig)
	assert.NoError(t, err)
	err = tests.VerifyIPs(wgConfig, verifyIPs, sk)
	assert.NoError(t, err)
	defer func() {
		_, err := tests.DownWG()
		assert.NoError(t, err)
	}()

	// Check that env variables set successfully
	res, err := tests.RemoteRun("root", pIP, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
