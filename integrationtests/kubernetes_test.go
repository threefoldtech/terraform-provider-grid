//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"strings"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestKubernetesDeployment(t *testing.T) {
	/* Test case for deployeng a k8s.

	   **Test Scenario**

	   - Deploy a k8s.
	   - Check that the outputs not empty.
	   - Check that master is reachable
	   - Check workers deployed number.
	   - Check that workers is ready.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Log(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./kubernetes",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	masterIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, masterIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	// Check that master is reachable
	err = tests.TestConnection(masterIP, "22")
	assert.NoError(t, err)

	// ssh to master node
	output, err := tests.RemoteRun("root", masterIP, "kubectl get node", sk)
	assert.NoError(t, err)
	output = strings.Trim(output, "\n")

	// // Check worker deployed number
	nodes := strings.Split(string(output), "\n")[1:]
	assert.Equal(t, 2, len(nodes)) // assert that there are 1 worker and 1 master

	// Check that worker is ready
	for i := 0; i < len(nodes); i++ {
		assert.Contains(t, nodes[i], "Ready")
	}

}
