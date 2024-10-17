//go:build integration
// +build integration

// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestGatewayPrivate(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("gateway_name_private", func(t *testing.T) {
		/* Test case for deploying a gateway name proxy.

		   **Test Scenario**

		   - Deploy a vm.
		   - Deploy a gateway name on the vm private network.
		   - Assert deployments outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_name_private",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		if err != nil &&
			(strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) ||
				strings.Contains(err.Error(), "error creating threefold plugin client")) {
			t.Skip("couldn't find any available nodes")
			return
		}

		require.NoError(t, err)

		// Check that the outputs not empty
		fqdn := terraform.Output(t, terraformOptions, "fqdn")
		require.NotEmpty(t, fqdn)

		myceliumIP := terraform.Output(t, terraformOptions, "mycelium_ip")
		require.NotEmpty(t, myceliumIP)

		ok := TestConnection(myceliumIP, "22")
		require.True(t, ok)

		_, err = RemoteRun("root", myceliumIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		require.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("https://%s", fqdn))
		require.NoError(t, err)

		if response != nil {
			body, err := io.ReadAll(response.Body)
			if body != nil {
				defer response.Body.Close()
			}
			require.NoError(t, err)
			require.Contains(t, string(body), "Directory listing for")
		}
	})

	t.Run("gateway_fqdn_private", func(t *testing.T) {
		/* Test case for deploying a gateway with FQDN.

		   **Test Scenario**

		   - Deploy a vm.
		   - Deploy a gateway with FQDN on the vm private network.
		   - Assert that outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		// make sure the test runs only on devnet
		if network, _ := os.LookupEnv("NETWORK"); network != "dev" {
			t.Skip()
			return
		}

		fqdn := "hamada1.3x0.me" // points to node 15 devnet

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_with_fqdn_private",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"fqdn":       fqdn,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		if err != nil &&
			(strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) ||
				strings.Contains(err.Error(), "error creating threefold plugin client")) {
			t.Skip("couldn't find any available nodes")
			return
		}

		require.NoError(t, err)

		// Check that the outputs not empty
		fqdn = terraform.Output(t, terraformOptions, "fqdn")
		require.NotEmpty(t, fqdn)

		myceliumIP := terraform.Output(t, terraformOptions, "mycelium_ip")
		require.NotEmpty(t, myceliumIP)

		ok := TestConnection(myceliumIP, "22")
		require.True(t, ok)

		_, err = RemoteRun("root", myceliumIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		require.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("https://%s", fqdn))
		require.NoError(t, err)

		if response != nil {
			body, err := io.ReadAll(response.Body)
			if body != nil {
				defer response.Body.Close()
			}
			require.NoError(t, err)
			require.Contains(t, string(body), "Directory listing for")
		}
	})
}
