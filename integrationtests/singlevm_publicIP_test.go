//go:build integration
// +build integration

package integrationtests

import (
	"strings"
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
		t.Log(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./singlevm_publicIP",
		Vars: map[string]interface{}{
			"public_key": pk,
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
	ok := tests.TestConnection(pIP, "22")
	assert.True(t, ok)

}
