package integrationtests

import (
	"context"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	gridproxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestNomad(t *testing.T) {
	publicKey, _, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("nomad_using_module", func(t *testing.T) {
		/* Test case for deployeng a nomad module.

		   **Test Scenario**

		   - Deploy a nomad with modules with three servers and one client.
		   - Make sure servers and client deployed and ready.

		*/

		first_server_ip := "10.1.2.2"
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

		// girdproxy tries to find 5 different nodes with suitable resources for the cluster
		cl := gridproxy.NewClient(url)
		status := "up"
		freeMRU := uint64(1024)
		freeSRU := uint64(1024)
		freeCRU := uint64(2)
		twinID := uint64(4653)

		f := types.NodeFilter{Status: &status, FreeMRU: &freeMRU, FreeSRU: &freeSRU, TotalCRU: &freeCRU, AvailableFor: &twinID}
		l := types.Limit{Page: 1, Size: 4, Randomize: true}

		res, _, err := cl.Nodes(context.Background(), f, l)
		if err != nil || len(res) < 4 {
			t.Fatal("gridproxy could not find nodes with suitable resources")
		}

		server1Node := res[0].NodeID
		server2Node := res[1].NodeID
		server3Node := res[2].NodeID
		client1Node := res[3].NodeID

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./nomad_using_module",
			Vars: map[string]interface{}{
				"ssh_key":         publicKey,
				"first_server_ip": first_server_ip,
				"network": map[string]interface{}{
					"name":        "nomadTestNetwork",
					"nodes":       []int{server1Node, server2Node, server3Node, client1Node},
					"ip_range":    "10.1.0.0/16",
					"description": "new network for nomad",
				},
				"servers": []map[string]interface{}{
					{
						"name":        "server1",
						"node":        server1Node,
						"cpu":         2,
						"memory":      1024,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server1dsk",
							"size": 5,
						},
					},
					{
						"name":        "server2",
						"node":        server2Node,
						"cpu":         2,
						"memory":      1024,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server2dsk",
							"size": 5,
						},
					},
					{
						"name":        "server3",
						"node":        server3Node,
						"cpu":         2,
						"memory":      1024,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server3dsk",
							"size": 5,
						},
					},
				},
				"clients": []map[string]interface{}{
					{
						"name":        "client1",
						"node":        client1Node,
						"cpu":         2,
						"memory":      1024,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "clientDisk",
							"size": 5,
						},
					},
				},
			},
		})

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		defer terraform.Destroy(t, terraformOptions)
		if !assert.NoError(t, err) {
			return
		}

		// Check that the outputs not empty
		server1IP := terraform.Output(t, terraformOptions, "server1_ip")
		assert.NotEmpty(t, server1IP)
		assert.Equal(t, server1IP, first_server_ip)

		server1YggIP := terraform.Output(t, terraformOptions, "server1_ygg_ip")
		assert.NotEmpty(t, server1YggIP)

		server2YggIP := terraform.Output(t, terraformOptions, "server2_ygg_ip")
		assert.NotEmpty(t, server2YggIP)

		server3YggIP := terraform.Output(t, terraformOptions, "server3_ygg_ip")
		assert.NotEmpty(t, server3YggIP)

		client1YggIP := terraform.Output(t, terraformOptions, "client1_ygg_ip")
		assert.NotEmpty(t, client1YggIP)

		// testing connection on port 22, waits at max 3mins until it becomes ready otherwise it fails
		ok = TestConnection(server1YggIP, "22")
		assert.True(t, ok)

		ok = TestConnection(server2YggIP, "22")
		assert.True(t, ok)

		ok = TestConnection(server3YggIP, "22")
		assert.True(t, ok)

		ok = TestConnection(client1YggIP, "22")
		assert.True(t, ok)
	})
}
