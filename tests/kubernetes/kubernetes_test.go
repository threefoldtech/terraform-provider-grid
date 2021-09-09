package test

import (
	"testing"

	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"strings"
	"os"
	"os/exec"
)

func TestKubernetesDeployment(t *testing.T) {
	    /* Test case for deployeng a k8s.

        **Test Scenario**

        - Deploy a k8s.
        - Check that the outputs not empty.
        - Up wireguard.
        - Check that master is reachable
        - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	public_key := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": public_key,
		  },
	})

	terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)
	defer tests.DownWG()

	// Check that the outputs not empty
	master_public_ip := terraform.Output(t, terraformOptions, "master_public_ip")
	assert.NotEmpty(t, master_public_ip)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)

	// Check that master is reachable
	out, _ := exec.Command("ping", master_public_ip, "-c 5", "-i 3", "-w 10").Output()
	assert.NotContains(t, string(out), "Destination Host Unreachable")

	// ssh to master node
	master_ip := strings.Split(master_public_ip, "/")[0]
	res, errors := tests.RemoteRun("root", master_ip, "kubectl get node")
	assert.Empty(t, errors)
	assert.NotEmpty(t, res)
}
