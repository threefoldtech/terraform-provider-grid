package test

import (
	"github.com/ashraffouda/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestSingleMountDeployment(t *testing.T) {
	/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

	   **Test Scenario**

	   - Deploy a mount disk with size 1G.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - ssh to VM and try to create a file with size 1G.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism:  1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err := terraform.InitAndApplyE(t, terraformOptions)

	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	pIP := strings.Split(publicIP, "/")[0]
	res, err := tests.RemoteRun("root", pIP, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt")
	assert.Empty(t, err)
	assert.Contains(t, string(res), "out of memory")
}
