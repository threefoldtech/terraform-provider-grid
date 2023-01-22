//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
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
	publicKey, _, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./wireguard",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	wgVM1IP := terraform.Output(t, terraformOptions, "vm1_wg_ip")
	assert.NotEmpty(t, wgVM1IP)

	wgVM2IP := terraform.Output(t, terraformOptions, "vm2_wg_ip")
	assert.NotEmpty(t, wgVM2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

	tempDir := t.TempDir()
	conf, err := UpWg(wgConfig, tempDir)
	assert.NoError(t, err)

	defer DownWG(conf)

	ips := []string{wgVM1IP, wgVM2IP}
	for i := range ips {
		// testing connection
		ok := TestConnection(ips[i], "22")
		assert.True(t, ok)
	}
}
