//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"os/exec"
	"testing"
	"time"

	"strings"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestKubernetesDeployment(t *testing.T) {
	/* Test case for deployeng a k8s.

	   **Test Scenario**

	   - Deploy a k8s.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - Check that master is reachable
	   - Check workers deployed number.
	   - Check that workers is ready.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./kubernetes",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	masterPublicIP := terraform.Output(t, terraformOptions, "computed_master_public_ip")
	assert.NotEmpty(t, masterPublicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	_, err = tests.UpWg(wgConfig)
	assert.NoError(t, err)
	defer func() {
		_, err := tests.DownWG()
		assert.NoError(t, err)
	}()

	// Check that master is reachable
	masterIP := strings.Split(masterPublicIP, "/")[0]
	err = tests.Wait(masterIP, "22")
	assert.NoError(t, err)

	out, _ := exec.Command("ping", masterPublicIP, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to master node
	time.Sleep(30 * (time.Second))
	res, err := tests.RemoteRun("root", masterIP, "kubectl get node", sk)
	assert.NoError(t, err)
	res = strings.Trim(res, "\n")

	// Check worker deployed number
	nodes := strings.Split(string(res), "\n")[1:]
	assert.Equal(t, 2, len(nodes)) // assert that there are 1 worker and 1 master

	// Check that worker is ready
	for i := 0; i < len(nodes); i++ {
		assert.Contains(t, nodes[i], "Ready")
	}
}
