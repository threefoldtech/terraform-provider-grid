package integrationtests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestWireguard(t *testing.T) {
	/* Test case for deploying a wireguard.

	   **Test Scenario**

	   - Deploy a wireguard.
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
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)
	if err != nil &&
		(strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) ||
			strings.Contains(err.Error(), "error creating threefold plugin client")) {
		t.Skip("couldn't find any available nodes")
		return
	}

	// Check that the outputs not empty
	wgVM1IP := terraform.Output(t, terraformOptions, "vm1_wg_ip")
	require.NotEmpty(t, wgVM1IP)

	wgVM2IP := terraform.Output(t, terraformOptions, "vm2_wg_ip")
	require.NotEmpty(t, wgVM2IP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	require.NotEmpty(t, wgConfig)

	tempDir := t.TempDir()
	conf, err := UpWg(wgConfig, tempDir)
	require.NoError(t, err)

	defer func() {
		_, err := DownWG(conf)
		require.NoError(t, err)
	}()

	ips := []string{wgVM1IP, wgVM2IP}
	for i := range ips {
		// testing connection
		require.True(t, TestConnection(ips[i], "22"))
	}
}
