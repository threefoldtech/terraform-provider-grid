//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestWireguard(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a singlenode.
	   - Check that the output is not empty.
	   - Up wireguard.
	   - Check that containers is reachable
	   - down wireguard
	   - Destroy the deployment

	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Log(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./wireguard",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	node1Container1IP := terraform.Output(t, terraformOptions, "vm1_wg_ip")
	assert.NotEmpty(t, node1Container1IP)

	node1Container2IP := terraform.Output(t, terraformOptions, "vm2_wg_ip")
	assert.NotEmpty(t, node1Container2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	conf, err := tests.UpWg(wgConfig, "", t)
	assert.NoError(t, err)
	defer tests.DownWG(conf)
	ips := []string{node1Container1IP, node1Container2IP}
	for i := range ips {
		err = tests.TestConnection(ips[i], "22")

		assert.NoError(t, err)
	}
}
