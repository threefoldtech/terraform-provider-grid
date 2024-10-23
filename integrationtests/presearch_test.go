//go:build integration
// +build integration

// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestPresearch(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a presearch.
	   - Check that the outputs not empty.
	   - Check that node is reachable.
	   - Check that vm is reachable
	   - Check that env variables set successfully
	   - Check prenode service is running
	   - Destroy the deployment
	*/
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	presearchRegistrationCode := "e5083a8d0a6362c6cf7a3078bfac81e3"
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./presearch",
		Vars: map[string]interface{}{
			"public_key":                  publicKey,
			"presearch_registration_code": presearchRegistrationCode,
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

	myceliumIP := terraform.Output(t, terraformOptions, "mycelium_ip")
	require.NotEmpty(t, myceliumIP)

	ok := TestConnection(myceliumIP, "22")
	require.True(t, ok)

	output, err := RemoteRun("root", myceliumIP, "cat /proc/1/environ", privateKey)
	require.NoError(t, err)
	require.Contains(t, string(output), fmt.Sprintf("PRESEARCH_REGISTRATION_CODE=%s", presearchRegistrationCode))

	ticker := time.NewTicker(2 * time.Second)
	for now := time.Now(); time.Since(now) < 1*time.Minute; {
		<-ticker.C
		output, err = RemoteRun("root", myceliumIP, "zinit list", privateKey)
		if err == nil && strings.Contains(output, "prenode: Success") {
			break
		}
	}

	require.NoError(t, err)
	require.Contains(t, output, "prenode: Success")
}
