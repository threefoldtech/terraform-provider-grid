package integrationtests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("vm", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode.
		   - Check that the outputs not empty.
		   - Check that vm is reachable.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

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

	t.Run("vm_public_ip", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode.
		   - Check that the outputs not empty.
		   - Check that vm is reachable
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_public_ip",
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

	t.Run("vm_invalid_cpu", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode with zero cpu.
		   - The deployment should fail
		   - Destroy the network

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_invalid_cpu",
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

	t.Run("vm_invalid_memory", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a singlenode with memory less than 250.
		   - The deployment should fail.
		   - Destroy the network.

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_invalid_memory",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		if err == nil {
			t.Errorf("Should fail with mem capacity can't be less that 250M but err is null")
		}

	})
	t.Run("vm_mounts", func(t *testing.T) {
		/* Test case for deployeng a disk and mount it.

		   **Test Scenario**

		   - Deploy a disk.
		   - Check that the outputs not empty.
		   - Verify the VMs ips.
		   - Check that disk has been mounted successfully with 10G.
		   - Destroy the deployment.

		*/

		// retryable errors in terraform testing.
		// generate ssh keys for test
		diskSize := 2
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_mounts",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"disk_size":  diskSize,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		node1Container1IP := terraform.Output(t, terraformOptions, "node1_container1_ip")
		assert.NotEmpty(t, node1Container1IP)

		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		assert.NotEmpty(t, wgConfig)

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// Check that disk has been mounted successfully
		output, err = RemoteRun("root", planetary, "df -h | grep -w /app", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), fmt.Sprintf("%dG", diskSize))
	})

	t.Run("vm_mount_invalid_write", func(t *testing.T) {
		/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

		   **Test Scenario**

		   - Deploy a mount disk with size 1G.
		   - Check that the outputs not empty.
		   - ssh to VM and try to create a file with size 1G.
		   - Destroy the deployment
		*/
		diskSize := 1
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_mount_invalid_write",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"disk_size":  diskSize,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// ssh to VM and try to create a file bigger than disk size.
		_, err = RemoteRun("root", planetary, fmt.Sprintf("cd /app/ && dd if=/dev/vda bs=%dG count=1 of=test.txt", diskSize+1), privateKey)
		if err == nil {
			t.Errorf("should fail with out of memory")
		}
	})
	t.Run("vm_multinode", func(t *testing.T) {
		/* Test case for deployeng a multinode.
		   **Test Scenario**
		   - Deploy multinode deployments.
		   - Check that the outputs not empty.
		   - Verify the VMs ips
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_multinode",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		node1Container1IP := terraform.Output(t, terraformOptions, "node1_zmachine1_ip")
		assert.NotEmpty(t, node1Container1IP)

		node2Container1IP := terraform.Output(t, terraformOptions, "node2_zmachine1_ip")
		assert.NotEmpty(t, node2Container1IP)

		yggIP1 := terraform.Output(t, terraformOptions, "node1_zmachine_ygg_ip")
		assert.NotEmpty(t, yggIP1)

		yggIP2 := terraform.Output(t, terraformOptions, "node2_zmachine_ygg_ip")
		assert.NotEmpty(t, yggIP2)

		//spliting ip to connect on it
		yggIP1 = strings.Split(yggIP1, "/")[0]

		yggIP2 = strings.Split(yggIP2, "/")[0]

		//testing connections
		ok := TestConnection(yggIP1, "22")
		assert.True(t, ok)

		ok = TestConnection(yggIP2, "22")
		assert.True(t, ok)
	})
}
