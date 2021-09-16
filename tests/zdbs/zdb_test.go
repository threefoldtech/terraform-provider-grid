package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grid-provider/tests"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestZdbsDeployment(t *testing.T) {
	/* Test case for deployeng a singlenode.

	   **Test Scenario**

	   - Deploy a zdbs.
	   - Deploy a VM (have a IPv6)
	   - Check that the outputs not empty.
	   - Check that zdb reachable from VM.
	   - Destroy the deployment

	*/

	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Parallelism:  1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	deploymentId := terraform.Output(t, terraformOptions, "deployment_id")
	assert.NotEmpty(t, deploymentId)

	zdb1Endpoint := terraform.Output(t, terraformOptions, "zdb1_endpoint")
	assert.NotEmpty(t, zdb1Endpoint)

	zdb1Namespace := terraform.Output(t, terraformOptions, "zdb1_namespace")
	assert.NotEmpty(t, zdb1Namespace)

	container1_ip := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, container1_ip)

	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	tests.UpWg(wgConfig)
	defer tests.DownWG()

	// install redis on vm
	_, err := tests.RemoteRun("root", container1_ip, "apk --update add redis")
	assert.Empty(t, err)

	redisIP := strings.Split(zdb1Endpoint[1:], "]")[0]
	redisPort := strings.Split(zdb1Endpoint, ":")[8]
	res, err1 := tests.RemoteRun("root", container1_ip, fmt.Sprintf("redis-cli -h %s -p %s ping", redisIP, redisPort))
	assert.Empty(t, err1)
	assert.Contains(t, res, "PONG")
}
