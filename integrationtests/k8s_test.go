package integrationtests

import (
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	gridproxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// AssertNodesAreReady runs `kubectl get node` on the master node and asserts that all nodes are ready
func AssertNodesAreReady(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, masterYggIP)

	output, err := RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	assert.Empty(t, err)

	nodesNumber := 2
	numberOfReadyNodes := strings.Count(output, "Ready")
	assert.True(t, numberOfReadyNodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %s nodes are ready", numberOfReadyNodes)

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
		assert.NoError(t, err)

		// Check that the outputs not empty
		masterIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, masterIP)

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
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a k8s with modules with one master and one worker.
		   - Make sure master and worker deployed and ready.
		   - Add one more worker.
		   - Make sure that worker is added and ready.

		*/

		gridProxyURL := map[string]string{
			"dev":  "https://gridproxy.dev.grid.tf/",
			"test": "https://gridproxy.test.grid.tf/",
			"qa":   "https://gridproxy.qa.grid.tf/",
			"main": "https://gridproxy.grid.tf/",
		}
		network := os.Getenv("NETWORK")
		if network == "" {
			network = "dev"
		}
		url, ok := gridProxyURL[network]
		if !ok {
			t.Fatalf("invalid network name %s", network)
		}
		// girdproxy tries to find 3 different nodes with suitable resources for the cluster
		cl := gridproxy.NewClient(url)
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
		l := types.Limit{
			Page: 1,
			Size: 3,
		}
		res, _, err := cl.Nodes(f, l)
		if err != nil || len(res) != 3 {
			t.Fatal("gridproxy could not find nodes with suitable resources")
		}

		masterNode := res[0].NodeID
		worker0Node := res[1].NodeID
		worker1Node := res[2].NodeID

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
		assert.NoError(t, err)

		AssertNodesAreReady(t, terraformOptions, privateKey)

	})
}
