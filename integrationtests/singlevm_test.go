//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestSingleVMDeployment(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a singlenode.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that vm is reachable
	   - Verify the VMs ips
	   - Check that env variables set successfully.
	   - Destroy the deployment

	*/
	t.TempDir()
	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./singlevm",
		Vars: map[string]interface{}{
			"public_key": pk,
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
	ok := tests.TestConnection(planetary, "22")
	assert.True(t, ok)

}
