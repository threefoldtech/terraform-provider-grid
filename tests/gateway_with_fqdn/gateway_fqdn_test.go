package test

import (
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSingleNodeDeployment(t *testing.T) {
	/* Test case for deployeng a gateway with fdqn.

	   **Test Scenario**

	   - Deploy a gateway with fdqn.
	   - Check that the outputs not empty.
	   - Check that ygg ip is reachable.
	   - Check that gateway point to backend.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// name := tests.RandomName()
	backend := "http://69.164.223.208:443"
	fqdn := "remote.hassan.grid.tf" // "remote." + name + ".grid.tf"

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"fqdn":    fqdn,
			"backend": backend,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	fqdn_ := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, fqdn_)

	out, _ := exec.Command("ping", fqdn_, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")
}
