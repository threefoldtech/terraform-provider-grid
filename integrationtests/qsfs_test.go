//go:build integration
// +build integration

// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestQSFS(t *testing.T) {
	if network, _ := os.LookupEnv("NETWORK"); network == "test" || network == "main" {
		t.Skip("https://github.com/threefoldtech/terraform-provider-grid/issues/770")
		return
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("qsfs_test", func(t *testing.T) {
		/* Test case for deploying a QSFS check metrics.
		   **Test Scenario**
		   - Deploy a qsfs.
		   - Check that the outputs not empty.
		   - Assert that qsfs create syscalls from metrics endpoint are equal to 0
		   - Write a file on qsfs
		   - Assert that qsfs create syscalls from metrics endpoint are equal to 1
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./qsfs",
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

		require.NoError(t, err)

		// Check that the outputs not empty
		metrics := terraform.Output(t, terraformOptions, "metrics")
		require.NotEmpty(t, metrics)

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		require.NotEmpty(t, yggIP)

		// get metrics
		cmd := exec.Command("curl", metrics)
		output, err := cmd.Output()
		require.NoError(t, err)
		require.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 0")

		// try write to a file in mounted disk
		_, err = RemoteRun("root", yggIP, "cd /qsfs && echo hamadatext >> hamadafile", privateKey)
		require.NoError(t, err)

		time.Sleep(5 * time.Second)

		// get metrics after write
		cmd = exec.Command("curl", metrics)
		output, err = cmd.Output()
		require.NoError(t, err)
		require.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 1")
	})
}
