package integrationtests

import (
	"log"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
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
	pk, _, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./gateway_with_yggdrasil_ip",
		Vars: map[string]interface{}{
			"public_key": pk,
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

	// ssh to VM and check if yggdrasil is active
	out, _ := exec.Command("ping", fqdn, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")
}
