//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/integrationtests"
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
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./multinode",
		Vars: map[string]interface{}{
			"public_key": pk,
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

	isIPReachable := []string{node1Container1IP, node1Container1IP}
	_, err = tests.UpWg(wgConfig)
	assert.NoError(t, err)
	err = tests.isIPReachable(wgConfig, isIPReachable, sk)
	assert.NoError(t, err)
	defer func() {
		_, err := tests.DownWG()
		assert.NoError(t, err)
	}()

	pIP := strings.Split(publicIP, "/")[0]
	err = tests.Wait(pIP, "22")
	assert.NoError(t, err)

	// Check that env variables set successfully
	res, err := tests.RemoteRun("root", pIP, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
