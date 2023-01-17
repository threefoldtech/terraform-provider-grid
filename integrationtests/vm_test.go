package integrationtests

import (
	"log"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	publicKey, _, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("single_vm", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode.
		   - Check that the outputs not empty.
		   - Check that vm is reachable.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./singlevm",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty

		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
		assert.NotEmpty(t, node1Container1IP)

		node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
		assert.NotEmpty(t, node1Container2IP)

		// Up wireguard
		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		assert.NotEmpty(t, wgConfig)

		// testing connection
		ok := TestConnection(planetary, "22")
		assert.True(t, ok)
	})

	t.Run("singleVM_WithPublicIP", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode.
		   - Check that the outputs not empty.
		   - Check that vm is reachable
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./singlevm_publicIP",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty

		publicip := terraform.Output(t, terraformOptions, "computed_public_ip")
		assert.NotEmpty(t, publicip)

		node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
		assert.NotEmpty(t, node1Container1IP)

		node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
		assert.NotEmpty(t, node1Container2IP)

		//spliting ip to connect on it
		pIP := strings.Split(publicip, "/")[0]

		//testing connections
		ok := TestConnection(pIP, "22")
		assert.True(t, ok)

	})

	t.Run("singleVM_WithKey", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode.
		   - Check that the outputs not empty.
		   - connect to the machine
		   - Destroy the deployment

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./singlevm_key_ed25519",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
		assert.NotEmpty(t, node1Container1IP)

		node1Container2IP := terraform.Output(t, terraformOptions, "node1_container2_ip")
		assert.NotEmpty(t, node1Container2IP)

		ok := TestConnection(planetary, "22")
		assert.True(t, ok)
	})

	t.Run("singleVM_WithZeroCPU", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode with zero cpu.
		   - The deployment should fail
		   - Destroy the network

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./singlevm_with_zero_cpu",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)

		if err == nil {
			t.Errorf("Should fail with can't deploy with 0 cpu but err is null")
		}
	})

	t.Run("singleVM_WithMemoryLessThan250", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode with memory less than 250.
		   - The deployment should fail.
		   - Destroy the network.

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./singlevm_with_memory_less_than_250M",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)

		if err == nil {
			t.Errorf("Should fail with mem capacity can't be less that 250M but err is null")
		}

	})

}
