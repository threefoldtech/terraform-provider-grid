package test

import (
	"testing"

	"os"
	"os/exec"
	"strings"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// type worker struct {
// 	disk_size int
// 	node      int
// 	name      string
// 	cpu       int
// 	memory    int
// }

func TestKubernetesDeployment(t *testing.T) {
	/* Test case for deployeng a k8s.

	   **Test Scenario**

	   - Deploy a k8s.
	   - Check that the outputs not empty.
	   - Redeploy k8s with anther worker.
	   - Up wireguard.
	   - Check that anther worker added successfully.
	   - Destroy the deployment.
	*/

	// retryable errors in terraform testing.
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
			"worker": map[string]interface{}{
				"disk_size": 15,
				"node":      2,
				"name":      "w1",
				"cpu":       20,
				"memory":    2048,
			},
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	masterPublicIP := terraform.Output(t, terraformOptions, "master_public_ip")
	assert.NotEmpty(t, masterPublicIP)

	// Redeploy k8s with anther worker
	terraformOptions1 := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
			"worker": map[string]interface{}{
				"disk_size": 15,
				"node":      2,
				"name":      "w2",
				"cpu":       20,
				"memory":    2048,
			},
		},
		Parallelism: 1,
	})
	terraform.InitAndApply(t, terraformOptions1)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions1, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// Check that master is reachable
	out, _ := exec.Command("ping", masterPublicIP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to master node
	masterIP := strings.Split(masterPublicIP, "/")[0]
	res, errors := tests.RemoteRun("root", masterIP, "kubectl get node")
	assert.Empty(t, errors)

	// Check worker deployed number
	nodes := strings.Split(res, "\n")
	workers := nodes[1:]               // remove header
	assert.Equal(t, len(workers)-1, 4) // assert that there are 3 workers and master
}
