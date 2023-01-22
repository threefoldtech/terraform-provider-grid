package integrationtests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a multinode.
	   **Test Scenario**
	   - Deploy a multinode.
	   - Check that the outputs not empty.
	   - Verify the VMs ips
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	publicKey, _, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./multinode",
		Vars: map[string]interface{}{
			"public_key": publicKey,
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

}
