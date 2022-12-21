package test

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestMountWithBiggerFileDeployment(t *testing.T) {
	/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

	   **Test Scenario**

	   - Deploy a mount disk with size 1G.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - ssh to VM and try to create a file with size 1G.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SshKeys()
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApplyE(t, terraformOptions)

	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// ssh to VM and try to create a file with size 1G.
	publicIP, err := tests.IPFromCidr(publicIP)
	assert.NoError(t, err)

	err = tests.Wait(publicIP, "22")
	assert.NoError(t, err)

	_, err = tests.RemoteRun("root", publicIP, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt")
	if err == nil {
		t.Errorf("should fail with out of memory")
	}
}
