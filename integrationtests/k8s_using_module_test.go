package integrationtests

import (
	"context"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestModuleK8s(t *testing.T) {
	/* Test case for deploying a k8s.

	   **Test Scenario**

	   - Deploy a k8s with modules with one master and one worker.
	   - Make sure master and worker deployed and ready.
	   - Add one more worker.
	   - Make sure that worker is added and ready.

	*/

	// t.Skip("https://github.com/threefoldtech/terraform-provider-grid/issues/770")

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

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

	nodes, err := deployer.FilterNodes(context.Background(), tfPlugin, f, []uint64{freeSRU}, []uint64{}, []uint64{})
	require.NoError(t, err)
	if len(nodes) < 3 {
		t.Skip("couldn't find enough nodes")
	}

	masterNode := nodes[0].NodeID
	worker0Node := nodes[1].NodeID
	worker1Node := nodes[2].NodeID

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./k8s_using_module",
		Vars: map[string]interface{}{
			"ssh":           publicKey,
			"network_nodes": []int{masterNode, worker0Node, worker1Node},
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

	RequireNodesAreReady(t, terraformOptions, privateKey)

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

	RequireNodesAreReady(t, terraformOptions, privateKey)
}
