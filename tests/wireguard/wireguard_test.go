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
	pk, sk, err := tests.GenerateSSHKeyPair()
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

	_, err = tests.UpWg(wgConfig)
	assert.NoError(t, err)
	defer tests.DownWG()
	ips := []string{node1Container1IP, node1Container2IP}
	for i := range ips {
		out, err := exec.Command("ping", ips[i], "-c 5", "-i 3", "-w 10").Output()

		assert.True(t, !strings.Contains(string(out), "Destination Host Unreachable"))
		assert.NoError(t, err)
	}
	// Check that env variables set successfully
	res, err := tests.RemoteRun("root", pIP, "cat /proc/1/environ", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
