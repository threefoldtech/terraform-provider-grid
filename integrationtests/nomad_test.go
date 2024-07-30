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

func TestNomad(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("nomad_using_module", func(t *testing.T) {
		/* Test case for deploying a nomad module.

		   **Test Scenario**

		   - Deploy a nomad with modules with three servers and one client.
		   - Make sure servers and client deployed and ready.
		   - Check number of clients deployed.

		*/

		firstServerIP := "10.1.2.2"

		tf, err := setup()
		if err != nil {
			t.Fatalf("failed to get create tf plugin client: %s", err.Error())
		}

		nodes, err := deployer.FilterNodes(
			context.Background(),
			tf,
			types.NodeFilter{
				Status:  []string{"up"},
				FreeMRU: convertMBToBytes(256),
				FreeSRU: convertGBToBytes(2),
				// Freefarm
				FarmIDs: []uint64{1},
			},
			[]uint64{*convertGBToBytes(1)},
			nil,
			[]uint64{*convertGBToBytes(1)},
			5,
		)
		if err != nil {
			t.Skip("could not find nodes with suitable resources")
			return
		}

		server1Node := nodes[0].NodeID
		server2Node := nodes[1].NodeID
		server3Node := nodes[2].NodeID
		client1Node := nodes[3].NodeID
		client2Node := nodes[4].NodeID

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./nomad_using_module",
			Vars: map[string]interface{}{
				"ssh_key":         publicKey,
				"first_server_ip": firstServerIP,
				"network": map[string]interface{}{
					"name":        "nomadTestNetwork",
					"nodes":       []int{server1Node, server2Node, server3Node, client1Node, client2Node},
					"ip_range":    "10.1.0.0/16",
					"description": "new network for nomad",
				},
				"servers": []map[string]interface{}{
					{
						"name":        "server1",
						"node":        server1Node,
						"cpu":         2,
						"memory":      256,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server1dsk",
							"size": 1,
						},
					},
					{
						"name":        "server2",
						"node":        server2Node,
						"cpu":         2,
						"memory":      256,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server2dsk",
							"size": 1,
						},
					},
					{
						"name":        "server3",
						"node":        server3Node,
						"cpu":         2,
						"memory":      256,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "server3dsk",
							"size": 1,
						},
					},
				},
				"clients": []map[string]interface{}{
					{
						"name":        "client1",
						"node":        client1Node,
						"cpu":         2,
						"memory":      256,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "client1Disk",
							"size": 1,
						},
					},
					{
						"name":        "client2",
						"node":        client2Node,
						"cpu":         2,
						"memory":      256,
						"rootfs_size": 1024,
						"mount_point": "/mnt",
						"disk": map[string]interface{}{
							"name": "client2Disk",
							"size": 1,
						},
					},
				},
			},
		})

		_, err = terraform.InitAndApplyE(t, terraformOptions)
		defer terraform.Destroy(t, terraformOptions)
		require.NoError(t, err)

		// Check that the outputs not empty
		server1IP := terraform.Output(t, terraformOptions, "server1_ip")
		require.NotEmpty(t, server1IP)
		require.Equal(t, server1IP, firstServerIP)

		server1YggIP := terraform.Output(t, terraformOptions, "server1_ygg_ip")
		require.NotEmpty(t, server1YggIP)

		server2YggIP := terraform.Output(t, terraformOptions, "server2_ygg_ip")
		require.NotEmpty(t, server2YggIP)

		server3YggIP := terraform.Output(t, terraformOptions, "server3_ygg_ip")
		require.NotEmpty(t, server3YggIP)

		client1YggIP := terraform.Output(t, terraformOptions, "client1_ygg_ip")
		require.NotEmpty(t, client1YggIP)

		// testing connection on port 22, waits at max 3mins until it becomes ready otherwise it fails
		ok := TestConnection(server1YggIP, "22")
		require.True(t, ok)

		ok = TestConnection(server2YggIP, "22")
		require.True(t, ok)

		ok = TestConnection(server3YggIP, "22")
		require.True(t, ok)

		ok = TestConnection(client1YggIP, "22")
		require.True(t, ok)

		// until services are ready
		time.Sleep(30 * time.Second)

		output, err := RemoteRun("root", server1YggIP, "nomad node status", privateKey)
		require.Empty(t, err)
		require.Equal(t, 2, strings.Count(output, "ready"))
	})
}
