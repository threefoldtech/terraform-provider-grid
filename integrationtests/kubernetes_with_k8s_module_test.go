//go:build integration
// +build integration

package integrationtests

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestKubernetesDeployment(t *testing.T) {
	publicKey, privateKey, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

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

	terraform.InitAndApply(t, terraformOptions)
	assertDeploymentStatus(t, terraformOptions, privateKey)

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
	assertDeploymentStatus(t, terraformOptions, privateKey)
	terraform.Destroy(t, terraformOptions)
}

func assertDeploymentStatus(t *testing.T, terraformOptions *terraform.Options, privateKey string) {
	t.Helper()

	masterYggIP := terraform.Output(t, terraformOptions, "master_yggip")
	assert.NotEmpty(t, masterYggIP)

	time.Sleep(5 * time.Second)
	output, err := tests.RemoteRun("root", masterYggIP, "kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	assert.Empty(t, err)

	nodesNumber := reflect.ValueOf(terraformOptions.Vars["workers"]).Len() + 1
	numberOfReadynodes := strings.Count(output, "Ready")
	assert.True(t, numberOfReadynodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %s nodes are ready", numberOfReadynodes)

}
