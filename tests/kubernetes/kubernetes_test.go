package test

import (
	"testing"
	
	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"

	"os/exec"
)

func TestKubernetesDeployment(t *testing.T) {
	    /* Test case for deployeng a k8s.

        **Test Scenario**

        - Deploy a k8s.
        - Check that the outputs not empty.
        - Up wireguard.
        - Check that master is reachable
        - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
	})

	terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	// Check that the outputs not empty
	master_public_ip := terraform.Output(t, terraformOptions, "master_public_ip")
	assert.NotEmpty(t, master_public_ip)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)

	// Check that master is reachable
	out, _ := exec.Command("ping", master_public_ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to master node
	res, errors := tests.RemoteRun("root", master_public_ip, "kubectl get node")
	assert.Empty(t, errors)
	assert.NotEmpty(t, res)
}
