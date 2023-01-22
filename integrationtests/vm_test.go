package integrationtests

import (
	"log"
	"os/exec"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
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

	t.Run("single_vm_with_public_ip", func(t *testing.T) {
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

	t.Run("single_vm_with_key", func(t *testing.T) {
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

	t.Run("single_vm_with_zero_cpu", func(t *testing.T) {
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

	t.Run("single_vm_with_memory_less_than_250", func(t *testing.T) {
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
	t.Run("qsfs", func(t *testing.T) {
		/* Test case for deployeng a QSFS check metrics.
		   **Test Scenario**
		   - Deploy a qsfs.
		   - Check that the outputs not empty.
		   - Check that containers is reachable
		   - write to a file. number of syscalls for write should increase try open, read, create, rename or etc number of syscalls should increase
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs_check_metrics",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		assert.NotEmpty(t, metrics)

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		// get metrics
		cmd := exec.Command("curl", metrics)
		output, _ := cmd.Output()

		// try write to a file in mounted disk
		_, err = RemoteRun("root", yggIP, "cd /qsfs && echo test >> test", privateKey)
		assert.NoError(t, err)
		// get metrics after write
		cmd = exec.Command("curl", metrics)
		output_after_write, _ := cmd.Output()

		// check that syscalls for write should increase
		assert.NotEqual(t, output, output_after_write)
	})

	t.Run("qsfs_read_write", func(t *testing.T) {
		/* Test case for deployeng a QSFS.
		**Test Scenario**
		- Deploy a qsfs.
		- Check that the outputs not empty.
		- Check that containers is reachable
		- get the qsfs one and find its path and do some operations there you should can writing/reading/listing
		- Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs_read_write",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		assert.NotEmpty(t, metrics)

		ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, ygg_ip)

		output, err := RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test && cat test", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "test")
	})
	t.Run("mounts", func(t *testing.T) {
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
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./mounts",
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

		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		assert.NotEmpty(t, wgConfig)

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// Check that disk has been mounted successfully with 10G
		output, err = RemoteRun("root", planetary, "df -h | grep -w /app", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "10G")
	})

	t.Run("mounts_with_bigger_file", func(t *testing.T) {
		/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

		   **Test Scenario**

		   - Deploy a mount disk with size 1G.
		   - Check that the outputs not empty.
		   - ssh to VM and try to create a file with size 1G.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./mount_with_file_bigger_than_disk_size",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// ssh to VM and try to create a file with size 1G.
		_, err = RemoteRun("root", planetary, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", privateKey)
		if err == nil {
			t.Errorf("should fail with out of memory")
		}
	})
	t.Run("mounts_with_bigger_file", func(t *testing.T) {
		/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

		   **Test Scenario**

		   - Deploy a mount disk with size 1G.
		   - Check that the outputs not empty.
		   - ssh to VM and try to create a file with size 1G.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./mount_with_file_bigger_than_disk_size",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		terraform.InitAndApply(t, terraformOptions)

		planetary := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, planetary)

		// Check that env variables set successfully
		output, err := RemoteRun("root", planetary, "cat /proc/1/environ", privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "SSH_KEY")

		// ssh to VM and try to create a file with size 1G.
		_, err = RemoteRun("root", planetary, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", privateKey)
		if err == nil {
			t.Errorf("should fail with out of memory")
		}
	})
}
