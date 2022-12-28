//go:build integration
// +build integration

package test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestSingleNodeDeployment(t *testing.T) {
	/* Test case for deployeng a gateway with ygg ip.

	   **Test Scenario**

	   - Deploy a VM in single node.
	   - Deploy a gateway with ygg ip.
	   - Check that the outputs not empty.
	   - Check that ygg ip is reachable.
	   - Check that gateway point to backend.
	   - Destroy the deployment.
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SSHKeys()
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
	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	fqdn := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, fqdn)

	out, _ := exec.Command("ping", yggIP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to VM and check if yggdrasil is active
	status := false
	status = tests.Wait(yggIP, "22")
	if status == false {
		t.Errorf("ygg ip not reachable")
	}

	out1, _ := exec.Command("ping", fqdn, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out1), "Destination Host Unreachable")
}
