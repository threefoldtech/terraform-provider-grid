package integrationtests

import (
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

// RequireNodesAreReady runs `kubectl get node` on the master node and requires that all nodes are ready
func RequireNodesAreReady(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "mr_ygg_ip")
	require.NotEmpty(t, masterYggIP)

	time.Sleep(40 * time.Second)

	output, err := RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	require.Empty(t, err)

	nodesNumber := 2
	numberOfReadyNodes := strings.Count(output, "Ready")
	require.True(t, numberOfReadyNodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %d nodes are ready", numberOfReadyNodes)
}

func TestK8s(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("kubernetes", func(t *testing.T) {
		/* Test case for deploying a k8s.

		   **Test Scenario**

		   - Deploy a k8s cluster.
		   - Check that the outputs not empty.
		   - require that all nodes are ready.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./k8s",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if err != nil &&
			strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) &&
			strings.Contains(err.Error(), "error creating threefold plugin client") {
			t.Skip("couldn't find any available nodes")
			return
		}

		require.NoError(t, err)

		// Check that the outputs not empty
		masterIP := terraform.Output(t, terraformOptions, "mr_ygg_ip")
		require.NotEmpty(t, masterIP)

		workerIP := terraform.Output(t, terraformOptions, "worker_ygg_ip")
		require.NotEmpty(t, workerIP)

		// Check wireguard config in output
		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		require.NotEmpty(t, wgConfig)

		// Check that master and workers is reachable
		// testing connection on port 22, waits at max 3mins until it becomes ready otherwise it fails
		ok := TestConnection(masterIP, "22")
		require.True(t, ok)

		ok = TestConnection(workerIP, "22")
		require.True(t, ok)

		// ssh to master node
		RequireNodesAreReady(t, terraformOptions, privateKey)
	})

	t.Run("k8s_invalid_names", func(t *testing.T) {
		/* Test case for deploying a k8s.

		   **Test Scenario**

		   - Deploy a k8s with 2 workers having the same name.
		   - Check that the deployment failed.
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./k8s_invalid_names",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)
		_, err = terraform.InitAndApplyE(t, terraformOptions)
		if err == nil {
			t.Errorf("k8s workers and master must have unique names")
		}

		if err != nil &&
			strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) &&
			strings.Contains(err.Error(), "error creating threefold plugin client") {
			t.Skip("couldn't find any available nodes")
			return
		}
	})
}
