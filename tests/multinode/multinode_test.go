package test

import (
	"strings"
	"testing"

	"os"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a multinode.
	   **Test Scenario**
	   - Deploy a multinode.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that containers is reachable
	   - Verify the VMs ips
	   - Check that env variables set successfully
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
	node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1Container1IP)

	node2Container1IP := terraform.Output(t, terraformOptions, "node2_container1_ip")
	assert.NotEmpty(t, node2Container1IP)

	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	verifyIPs := []string{node1Container1IP, node1Container1IP}
	tests.VerifyIPs(wgConfig, verifyIPs)
	defer tests.DownWG()

	pIP := strings.Split(publicIP, "/")[0]
	status := false
	for i := 0; i < 100; i++ {
		status = tests.Wait(pIP, "22")
		if status {
			break
		}
	}
	if status == false {
		t.Errorf("ip not reachable")
	}

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", pIP, "cat /proc/1/environ")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
