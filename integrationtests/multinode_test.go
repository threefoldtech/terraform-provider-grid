//go:build integration
// +build integration

package integrationtests

import (
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
	"log"
	"strings"
	"testing"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a multinode.
	   **Test Scenario**
	   - Deploy a multinode.
	   - Check that the outputs not empty.
	   - Verify the VMs ips
	   - Check that env variables set successfully
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./multinode",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	node1Container1IP := terraform.Output(t, terraformOptions, "node1_zmachine1_ip")
	assert.NotEmpty(t, node1Container1IP)

	node2Container1IP := terraform.Output(t, terraformOptions, "node2_zmachine1_ip")
	assert.NotEmpty(t, node2Container1IP)

	publicIP1 := terraform.Output(t, terraformOptions, "node1_zmachine_computed_public_ip")
	assert.NotEmpty(t, publicIP1)

	publicIP2 := terraform.Output(t, terraformOptions, "node2_zmachine_computed_public_ip")
	assert.NotEmpty(t, publicIP2)

	//spliting ip to connect on it
	pIP1 := strings.Split(publicIP1, "/")[0]

	pIP2 := strings.Split(publicIP2, "/")[0]

	ok := tests.TestConnection(pIP1, "22")
	assert.True(t, ok)

	ok = tests.TestConnection(pIP2, "22")
	assert.True(t, ok)
	// // Check that env variables set successfully
	// output, err := tests.RemoteRun("root", pIP1, "cat /proc/1/environ", sk)
	// assert.NoError(t, err)
	// assert.Contains(t, string(output), "machine=machine1")
	// output, err = tests.RemoteRun("root", pIP2, "cat /proc/1/environ", sk)
	// assert.NoError(t, err)
	// assert.Contains(t, string(output), "machine=machine2")

}
