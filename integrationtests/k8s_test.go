package integrationtests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// AssertNodesAreReady runs `kubectl get node` on the master node and asserts that all nodes are ready
func AssertNodesAreReady(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "mr_ygg_ip")
	assert.NotEmpty(t, masterYggIP)

	time.Sleep(10 * time.Second)

	output, err := RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	assert.Empty(t, err)

	nodesNumber := 2
	numberOfReadyNodes := strings.Count(output, "Ready")
	assert.True(t, numberOfReadyNodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %d nodes are ready", numberOfReadyNodes)
}

func TestK8s(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("kubernetes", func(t *testing.T) {
		/* Test case for deployeng a k8s.

		   **Test Scenario**

		   - Deploy a k8s cluster.
		   - Check that the outputs not empty.
		   - Assert that all nodes are ready.
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
		if !assert.NoError(t, err) {
			return
		}

		// Check that the outputs not empty
		masterIP := terraform.Output(t, terraformOptions, "mr_ygg_ip")
		if !assert.NotEmpty(t, masterIP) {
			return
		}

		workerIP := terraform.Output(t, terraformOptions, "worker_ygg_ip")
		assert.NotEmpty(t, workerIP)

		// Check wireguard config in output
		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		assert.NotEmpty(t, wgConfig)

		// Check that master and workers is reachable
		// testing connection on port 22, waits at max 3mins until it becomes ready otherwise it fails
		ok := TestConnection(masterIP, "22")
		assert.True(t, ok)

		ok = TestConnection(workerIP, "22")
		assert.True(t, ok)

		// ssh to master node
		AssertNodesAreReady(t, terraformOptions, privateKey)
	})

	t.Run("k8s_invalid_names", func(t *testing.T) {
		/* Test case for deployeng a k8s.

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
		// t.Skip("https://github.com/threefoldtech/terraform-provider-grid/issues/770")
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a k8s with modules with one master and one worker.
		   - Make sure master and worker deployed and ready.
		   - Add one more worker.
		   - Make sure that worker is added and ready.

		*/

		mnemonics := os.Getenv("MNEMONICS")
		if mnemonics == "" {
			t.Fatal("invalid empty mnemonic")
		}

		network := os.Getenv("NETWORK")
		if network == "" {
			network = "dev"
		}

		tfPlugin, err := deployer.NewTFPluginClient(mnemonics, "sr25519", network, "", "", "", 0, false)
		assert.NoError(t, err)

		status := "up"
		freeMRU := uint64(1024)
		freeSRU := uint64(2 * 1024)
		freeCRU := uint64(1)
		f := types.NodeFilter{
			Status:   &status,
			FreeMRU:  &freeMRU,
			FreeSRU:  &freeSRU,
			TotalCRU: &freeCRU,
		}

		nodes, err := deployer.FilterNodes(context.Background(), tfPlugin, f, []uint64{freeSRU}, []uint64{}, []uint64{}, 3)
		if err != nil || len(nodes) != 3 {
			t.Fatal("gridproxy could not find nodes with suitable resources")
		}
		assert.NoError(t, err)

		masterNode := nodes[0].NodeID
		worker0Node := nodes[1].NodeID
		// worker1Node := nodes[2].NodeID

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
		assert.NoError(t, err)
		defer terraform.Destroy(t, terraformOptions)

		AssertNodesAreReady(t, terraformOptions, privateKey)

		// terraformOptions.Vars["workers"] = []map[string]interface{}{
		// 	{
		// 		"name":        "w0",
		// 		"node":        worker0Node,
		// 		"cpu":         1,
		// 		"memory":      1024,
		// 		"disk_name":   "w0disk",
		// 		"mount_point": "/mydisk",
		// 		"publicip":    false,
		// 		"planetary":   true,
		// 	},
		// 	{
		// 		"name":        "w1",
		// 		"node":        worker1Node,
		// 		"cpu":         1,
		// 		"memory":      1024,
		// 		"disk_name":   "w1disk",
		// 		"mount_point": "/mydisk",
		// 		"publicip":    false,
		// 		"planetary":   true,
		// 	},
		// }
		// terraformOptions.Vars["disks"] = []map[string]interface{}{
		// 	{
		// 		"name":        "mrdisk",
		// 		"node":        masterNode,
		// 		"size":        2,
		// 		"description": "",
		// 	},
		// 	{
		// 		"name":        "w0disk",
		// 		"node":        worker0Node,
		// 		"size":        2,
		// 		"description": "",
		// 	},
		// 	{
		// 		"name":        "w1disk",
		// 		"node":        worker1Node,
		// 		"size":        2,
		// 		"description": "",
		// 	},
		// }
		// _, err = terraform.ApplyE(t, terraformOptions)
		// assert.NoError(t, err)

		// AssertNodesAreReady(t, terraformOptions, privateKey)
	})
}
