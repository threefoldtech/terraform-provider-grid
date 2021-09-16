package test

import (
	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestSingleMountDeployment(t *testing.T) {
	/* Test case for deployeng a disk and mount it.

	   **Test Scenario**

	   - Deploy a disk.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that containers is reachable.
	   - Verify the VMs ips.
	   - Check that env variables set successfully
	   - Check that disk has been mounted successfully with 10G.
	   - Destroy the deployment.

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
	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1Container1IP)

	node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1Container2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// Check that containers is reachable
	out, _ := exec.Command("ping", publicIP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	out1, _ := exec.Command("ping", node1Container2IP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")

	out2, _ := exec.Command("ping", node1Container1IP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out2), "Destination Host Unreachable")

	// ssh to containers
	_, errors1 := tests.RemoteRun("root", node1Container1IP, "ls")
	assert.Empty(t, errors1)

	_, errors2 := tests.RemoteRun("root", node1Container2IP, "ls")
	assert.Empty(t, errors2)

	// Verify the VMs ips
	res_ip, _ := tests.RemoteRun("root", node1Container2IP, "ifconfig")
	assert.Contains(t, string(res_ip), node1Container2IP)

	res_ip1, _ := tests.RemoteRun("root", node1Container1IP, "ifconfig")
	assert.Contains(t, string(res_ip1), node1Container1IP)

	// Check that env variables set successfully
	res, _ := tests.RemoteRun("root", node1Container1IP, "printenv")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	// Check that disk has been mounted successfully with 10G
	res1, errors3 := tests.RemoteRun("root", node1Container2IP, "df -h | grep -w /app")
	assert.Empty(t, errors3)
	assert.Contains(t, string(res1), "10.0G")
}
