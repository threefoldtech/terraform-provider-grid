//go:build integration
// +build integration

package test

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestKubernetesDeployment(t *testing.T) {
	pk, sk, err := tests.SshKeys()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Parallelism:  1,
		Vars: map[string]interface{}{
			"ssh":           pk,
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

	terraform.InitAndApply(t, terraformOptions)
	assertDeploymentStatus(t, terraformOptions, sk)

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

	terraform.Apply(t, terraformOptions)
	assertDeploymentStatus(t, terraformOptions, sk)
	terraform.Destroy(t, terraformOptions)
}

func assertDeploymentStatus(t *testing.T, terraformOptions *terraform.Options, sk string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "master_yggip")
	assert.NotEmpty(t, masterYggIP)

	time.Sleep(5 * time.Second)
	res, err := tests.RemoteRun("root", masterYggIP, "kubectl get node", sk)
	res = strings.Trim(res, "\n")
	assert.Empty(t, err)

	nodes := strings.Split(res, "\n")[1:]

	for _, node := range nodes {
		assert.Contains(t, node, "Ready")
	}
}
