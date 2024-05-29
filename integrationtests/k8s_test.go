package integrationtests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// requireNodesAreReady runs `kubectl get node` on the master node and requires that all nodes are ready
func requireNodesAreReady(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "mr_ygg_ip")
	require.NotEmpty(t, masterYggIP)

	time.Sleep(10 * time.Second)

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
		requireNodesAreReady(t, terraformOptions, privateKey)
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
	})

	t.Run("k8s_using_module", func(t *testing.T) {
		t.Skip("https://github.com/threefoldtech/terraform-provider-grid/issues/770")
		/* Test case for deploying a k8s.

		   **Test Scenario**

		   - Deploy a k8s with modules with one master and one worker.
		   - Make sure master and worker deployed and ready.
		   - Add one more worker.
		   - Make sure that worker is added and ready.

		*/

		tfPlugin, err := setup()
		require.NoError(t, err)

		status := "up"
		freeMRU := uint64(1024)
		freeSRU := uint64(2 * 1024)
		freeCRU := uint64(1)
		f := types.NodeFilter{
			Status:   []string{status},
			FreeMRU:  &freeMRU,
			FreeSRU:  &freeSRU,
			TotalCRU: &freeCRU,
		}

		nodes, err := deployer.FilterNodes(context.Background(), tfPlugin, f, []uint64{freeSRU}, []uint64{}, []uint64{}, 3)
		if err != nil || len(nodes) != 3 {
			t.Fatal("grid proxy could not find nodes with suitable resources")
		}
		require.NoError(t, err)

		masterNode := nodes[0].NodeID
		worker0Node := nodes[1].NodeID
		worker1Node := nodes[2].NodeID

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./k8s_using_module",
			Vars: map[string]interface{}{
				"ssh":           publicKey,
				"network_nodes": []int{12, masterNode},
				"master": map[string]interface{}{
					"name":        "mr",
					"node":        masterNode,
					"cpu":         1,
					"memory":      1024,
					"disk_name":   "mrdisk",
					"mount_point": "/mydisk",
					"publicip":    false,
					"planetary":   true,
				},
				"workers": []map[string]interface{}{
					{
						"name":        "w0",
						"node":        worker0Node,
						"cpu":         1,
						"memory":      1024,
						"disk_name":   "w0disk",
						"mount_point": "/mydisk",
						"publicip":    false,
						"planetary":   true,
					},
				},
				"disks": []map[string]interface{}{
					{
						"name":        "mrdisk",
						"node":        masterNode,
						"size":        2,
						"description": "",
					},
					{
						"name":        "w0disk",
						"node":        worker0Node,
						"size":        2,
						"description": "",
					},
				},
			},
		})

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		require.NoError(t, err)
		defer terraform.Destroy(t, terraformOptions)

		requireNodesAreReady(t, terraformOptions, privateKey)

		terraformOptions.Vars["workers"] = []map[string]interface{}{
			{
				"name":        "w0",
				"node":        worker0Node,
				"cpu":         1,
				"memory":      1024,
				"disk_name":   "w0disk",
				"mount_point": "/mydisk",
				"publicip":    false,
				"planetary":   true,
			},
			{
				"name":        "w1",
				"node":        worker1Node,
				"cpu":         1,
				"memory":      1024,
				"disk_name":   "w1disk",
				"mount_point": "/mydisk",
				"publicip":    false,
				"planetary":   true,
			},
		}
		terraformOptions.Vars["disks"] = []map[string]interface{}{
			{
				"name":        "mrdisk",
				"node":        masterNode,
				"size":        2,
				"description": "",
			},
			{
				"name":        "w0disk",
				"node":        worker0Node,
				"size":        2,
				"description": "",
			},
			{
				"name":        "w1disk",
				"node":        worker1Node,
				"size":        2,
				"description": "",
			},
		}
		_, err = terraform.ApplyE(t, terraformOptions)
		require.NoError(t, err)

		requireNodesAreReady(t, terraformOptions, privateKey)
	})
}
