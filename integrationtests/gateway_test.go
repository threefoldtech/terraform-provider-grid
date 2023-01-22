package integrationtests

import (
	"log"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestGateWay(t *testing.T) {
	_, _, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("gateway_fqdn", func(t *testing.T) {
		/* Test case for deployeng a gateway with fdqn.

		   **Test Scenario**

		   - Deploy a gateway with fdqn.
		   - Check that the outputs not empty.
		   - Check that ygg ip is reachable.
		   - Check that gateway point to backend.
		   - Destroy the deployment
		*/

		backend := "http://69.164.223.208:443"
		fqdn := "remote.hassan.grid.tf" // "remote." + name + ".grid.tf"

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_with_fqdn",
			Vars: map[string]interface{}{
				"fqdn":    fqdn,
				"backend": backend,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		fqdn = terraform.Output(t, terraformOptions, "fqdn")
		assert.NotEmpty(t, fqdn)

		output, _ := exec.Command("ping", fqdn, "-c 5", "-i 3", "-w 10").Output()
		assert.NotContains(t, string(output), "Destination Host Unreachable")
	})

	t.Run("gateway_yggip", func(t *testing.T) {
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
		publicKey, _, err := GenerateSSHKeyPair()
		if err != nil {
			t.Fatal()
		}

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_with_yggdrasil_ip",
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

		// ssh to VM and check if yggdrasil is active
		output, _ := exec.Command("ping", fqdn, "-c 5", "-i 3", "-w 10").Output()
		assert.NotContains(t, string(output), "Destination Host Unreachable")
	})

}
