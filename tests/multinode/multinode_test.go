package test

import (
	"testing"

	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
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

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// Check that containers is reachable
	out1, _ := exec.Command("ping", node1Container1IP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")

	out2, _ := exec.Command("ping", node2Container1IP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out2), "Destination Host Unreachable")

	// ssh to container
	_, errors1 := tests.RemoteRun("root", node1Container1IP, "ls")
	assert.Empty(t, errors1)

	_, errors2 := tests.RemoteRun("root", node2Container1IP, "ls")
	assert.Empty(t, errors2)

	// Verify the VMs ips
	res_ip, _ := tests.RemoteRun("root", node2Container1IP, "ifconfig")
	assert.Contains(t, string(res_ip), node2Container1IP)

	res1_ip, _ := tests.RemoteRun("root", node1Container1IP, "ifconfig")
	assert.Contains(t, string(res1_ip), node1Container1IP)

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", node1Container1IP, "printenv")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
