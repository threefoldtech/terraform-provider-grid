package integrationtests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestVM(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("single_vm", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a single vm.
		   - Check that the outputs not empty.
		   - Check that vm is reachable.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if !(assert.NoError(t, err)) {
			return
		}

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		vmIP := terraform.Output(t, terraformOptions, "vm_ip")
		assert.NotEmpty(t, vmIP)

		// testing connection
		ok := TestConnection(yggIP, "22")
		assert.True(t, ok)
	})

	t.Run("vm_public_ip", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a vm with a public ip.
		   - Check that the outputs not empty.
		   - Check that vm is reachable through its public ip
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_public_ip",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if err != nil && strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) {
			t.Skip("couldn't find any nodes with public ip")
			return
		}
		if !(assert.NoError(t, err)) {
			return
		}

		vmComputedIP := terraform.Output(t, terraformOptions, "vm_public_ip")
		assert.NotEmpty(t, vmComputedIP)

		vmIP := terraform.Output(t, terraformOptions, "vm_ip")
		assert.NotEmpty(t, vmIP)

		// spliting ip to connect on it
		publicIP := strings.Split(vmComputedIP, "/")[0]

		// testing connections
		ok := TestConnection(publicIP, "22")
		assert.True(t, ok)
	})

	t.Run("vm_invalid_cpu", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a vm with invalid cpu (0).
		   - The deployment should fail.
		   - Destroy the deployment.

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_invalid_cpu",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)

		assert.Error(t, err, "Should fail with can't deploy with 0 cpu but err is null")
	})

	t.Run("vm_invalid_memory", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a vm with memory less than 256.
		   - The deployment should fail.
		   - Destroy the deployment.

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_invalid_memory",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.Error(t, err, "Should fail with mem capacity can't be less that 250M but err is null")
	})
	t.Run("vm_mounts", func(t *testing.T) {
		/* Test case for deployeng a disk and mount it.

		   **Test Scenario**

		   - Deploy a vm mouting a disk.
		   - Check that the outputs are not empty.
		   - Check that disk has been mounted successfully.
		   - Destroy the deployment.

		*/

		// retryable errors in terraform testing.
		// generate ssh keys for test
		diskSize := 2
		mountPoint := "app"
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_mounts",
			Vars: map[string]interface{}{
				"public_key":  publicKey,
				"disk_size":   diskSize,
				"mount_point": mountPoint,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if !(assert.NoError(t, err)) {
			return
		}

		// Check that the outputs not empty
		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		vmIP := terraform.Output(t, terraformOptions, "vm_ip")
		assert.NotEmpty(t, vmIP)

		ok := TestConnection(yggIP, "22")
		assert.True(t, ok)

		// Check that disk has been mounted successfully
		output, err := RemoteRun("root", yggIP, fmt.Sprintf("df -h | grep -w /%s", mountPoint), privateKey)
		assert.NoError(t, err)
		assert.Contains(t, string(output), fmt.Sprintf("%d.0G", diskSize))
	})

	t.Run("vm_mount_invalid_write", func(t *testing.T) {
		/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

		   **Test Scenario**

		   - Deploy a vm mounting a disk.
		   - Check that the outputs not empty.
		   - ssh to VM and try to create a file with size larger than the disk size.
		   - Destroy the deployment
		*/
		diskSize := 1
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_mount_invalid_write",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"disk_size":  diskSize,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if !(assert.NoError(t, err)) {
			return
		}

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		ok := TestConnection(yggIP, "22")
		assert.True(t, ok)

		// ssh to VM and try to create a file bigger than disk size.
		_, err = RemoteRun("root", yggIP, fmt.Sprintf("cd /app/ && dd if=/dev/vda bs=%dG count=1 of=test.txt", diskSize+1), privateKey)
		assert.Error(t, err, "should fail with out of memory")
	})
	t.Run("vm_multinode", func(t *testing.T) {
		/* Test case for deployeng a multinode.
		   **Test Scenario**

		   - Deploy two vms on multiple nodes.
		   - Check that the outputs are not empty.
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./vm_multinode",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if !(assert.NoError(t, err)) {
			return
		}

		// Check that the outputs not empty
		vm1IP := terraform.Output(t, terraformOptions, "vm1_ip")
		assert.NotEmpty(t, vm1IP)

		vm2IP := terraform.Output(t, terraformOptions, "vm2_ip")
		assert.NotEmpty(t, vm2IP)

		vm1YggIP := terraform.Output(t, terraformOptions, "vm1_ygg_ip")
		assert.NotEmpty(t, vm1YggIP)

		vm2YggIP := terraform.Output(t, terraformOptions, "vm2_ygg_ip")
		assert.NotEmpty(t, vm2YggIP)

		// testing connections
		ok := TestConnection(vm1YggIP, "22")
		assert.True(t, ok)

		ok = TestConnection(vm2YggIP, "22")
		assert.True(t, ok)
	})
}
