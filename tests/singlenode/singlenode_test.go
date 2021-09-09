package test

import (
	"testing"
    "github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
)

func TestSingleNodeDeployment(t *testing.T) {
    /* Test case for deployeng a singlenode.

        **Test Scenario**

        - Deploy a singlenode.
        - Check that the outputs not empty.
        - Up wireguard.
        - Check that containers is reachable
        - Destroy the deployment

    */

	// retryable errors in terraform testing.
	public_key := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": public_key,
		  },
		Parallelism: 1,
	})

	terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)
	defer tests.DownWG()

    // Check that the outputs not empty
	public_ip := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, public_ip)

    node1_container1_ip := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1_container1_ip)

    node1_container2_ip := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1_container2_ip)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	
	// Check that containers is reachable
	out, _ := exec.Command("ping", public_ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")
	
	out1, _ := exec.Command("ping", node1_container2_ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")

	out2, _ := exec.Command("ping", node1_container1_ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out2), "Destination Host Unreachable")

    // ssh to container
	_, errors := tests.RemoteRun("root", node1_container1_ip, "ls")
	assert.Empty(t, errors)
}
