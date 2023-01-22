package integrationtests

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// AssertNodesAreReady runs `kubectl get node` on the master node and asserts that all nodes are ready
func AssertNodesAreReady(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "master_yggip")
	assert.NotEmpty(t, masterYggIP)

	time.Sleep(5 * time.Second)
	output, err := RemoteRun("root", masterYggIP, "kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	assert.Empty(t, err)

	nodesNumber := reflect.ValueOf(terraformOptions.Vars["workers"]).Len() + 1
	numberOfReadynodes := strings.Count(output, "Ready")
	assert.True(t, numberOfReadynodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %s nodes are ready", numberOfReadynodes)

}
func TestK8s(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("k8s", func(t *testing.T) {
		/* Test case for deployeng a k8s.

		   **Test Scenario**

		   - Deploy a k8s cluster.
		   - Check that the outputs not empty.
		   - Assert that all nodes are ready.
		   - Destroy the deployment
		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./kubernetes",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		masterIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, masterIP)

		// Check wireguard config in output
		wgConfig := terraform.Output(t, terraformOptions, "wg_config")
		assert.NotEmpty(t, wgConfig)

		// Check that master is reachable
		// testing connection on port 22, waits at max 3mins until it becomes ready otherwise it fails
		ok := TestConnection(masterIP, "22")
		assert.True(t, ok)

		// ssh to master node
		AssertNodesAreReady(t, terraformOptions, privateKey)
	})

	t.Run("k8s_with_2workers_same_name", func(t *testing.T) {
		/* Test case for deployeng a k8s.

		   **Test Scenario**

		   - Deploy a k8s with 2 workers having the same name.
		   - Check that the deployment failed.
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./kubernetes_with_2worker_same_name",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err = terraform.InitAndApplyE(t, terraformOptions)

		if err == nil {
			t.Errorf("k8s workers and master must have unique names")
		}
	})

	t.Run("k8s_with_module", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a k8s with modules with one master and one worker.
		   - Make sure master and worker deployed and ready.
		   - Add one more worker.
		   - Make sure that worker is added and ready.

		*/
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./kubernetes_with_k8s_module",
			Parallelism:  1,
			Vars: map[string]interface{}{
				"ssh":           publicKey,
				"network_nodes": []int{12, 219},
				"workers": []map[string]interface{}{
					{
						"name":        "w0",
						"node":        219,
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
						"node":        12,
						"size":        5,
						"description": "",
					},
					{
						"name":        "w0disk",
						"node":        219,
						"size":        2,
						"description": "",
					},
				},
			},
		})

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		AssertNodesAreReady(t, terraformOptions, privateKey)

		terraformOptions.Vars["workers"] = []map[string]interface{}{
			{
				"name":        "w0",
				"node":        219,
				"cpu":         1,
				"memory":      1024,
				"disk_name":   "w0disk",
				"mount_point": "/mydisk",
				"publicip":    false,
				"planetary":   true,
			},
			{
				"name":        "w1",
				"node":        12,
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
				"node":        12,
				"size":        5,
				"description": "",
			},
			{
				"name":        "w0disk",
				"node":        219,
				"size":        2,
				"description": "",
			},
			{
				"name":        "w1disk",
				"node":        12,
				"size":        2,
				"description": "",
			},
		}
		_, err = terraform.ApplyE(t, terraformOptions)
		assert.NoError(t, err)

		AssertNodesAreReady(t, terraformOptions, privateKey)

		terraform.Destroy(t, terraformOptions)
	})
}
