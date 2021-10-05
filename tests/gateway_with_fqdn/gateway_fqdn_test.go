package test

import (
	"os/exec"
	"testing"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
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
	name := tests.RandomName()
	backend := "http://69.164.223.208:443"
	fdqn := "remote." + name + ".grid.tf"
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"fdqn":    fdqn,
			"backend": backend,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	fqdn := terraform.Output(t, terraformOptions, "fqdn")
	assert.NotEmpty(t, fqdn)

	out, _ := exec.Command("ping", fqdn, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

}
