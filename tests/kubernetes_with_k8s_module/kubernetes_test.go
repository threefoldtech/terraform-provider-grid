package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestKubernetesDeployment(t *testing.T) {
	tests.SshKeys()
	sshKey := os.Getenv("PUBLICKEY")

    file, _ := os.Create("verbose.txt")
    defer file.Close()

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Parallelism:  1,
		Vars: map[string]interface{}{
			"ssh":           sshKey,
			"network_nodes": []int{45, 49},
			"workers": []map[string]interface{}{
				{
					"name":        "w0",
					"node":        49,
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
					"node":        45,
					"size":        5,
					"description": "",
				},
				{
					"name":        "w0disk",
					"node":        49,
					"size":        2,
					"description": "",
				},
			},
		},
	})

	terraform.InitAndApply(t, terraformOptions)
    assertDeploymentStatus(t, terraformOptions, file)

	terraformOptions.Vars["workers"] = []map[string]interface{} {
        {
            "name":        "w0",
            "node":        49,
            "cpu":         1,
            "memory":      1024,
            "disk_name":   "w0disk",
            "mount_point": "/mydisk",
            "publicip":    false,
            "planetary":   true,
        },
        {
            "name":        "w1",
            "node":        49,
            "cpu":         1,
            "memory":      1024,
            "disk_name":   "w1disk",
            "mount_point": "/mydisk",
            "publicip":    false,
            "planetary":   true,
        },
    }
	terraformOptions.Vars["disks"] = []map[string]interface{} {
        {
            "name":        "mrdisk",
            "node":        45,
            "size":        5,
            "description": "",
        },
        {
            "name":        "w0disk",
            "node":        49,
            "size":        2,
            "description": "",
        },
        {
            "name":        "w1disk",
            "node":        49,
            "size":        2,
            "description": "",
        },
    }

	terraform.Apply(t, terraformOptions)
    assertDeploymentStatus(t, terraformOptions, file)
	terraform.Destroy(t, terraformOptions)
}

func assertDeploymentStatus(t *testing.T, terraformOptions *terraform.Options, file *os.File) {
    t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "master_yggip")
	assert.NotEmpty(t, masterYggIP)
    fmt.Fprintln(file, masterYggIP)

	res, err := tests.RemoteRun("root", masterYggIP, "kubectl get node")
	assert.Empty(t, err)

    time.Sleep(3 * time.Second)
	nodes := strings.Split(res, "\n")[1:]
    fmt.Fprintf(file, "All nodes: %#v\n\n", nodes)

	for _, node := range nodes {
        fmt.Fprintln(file, node)
		assert.Contains(t, node, "Ready")
	}
}
