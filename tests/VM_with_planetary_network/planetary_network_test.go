package test

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestSingleNodeDeployment(t *testing.T) {
	/* Test case for deployeng a VM with planetary network.

	   **Test Scenario**

	   - Deploy a VM in single node.
	   - Check that the outputs not empty.
	   - Verify the VMs ips
	   - Check that ygg ip is reachable.
	   - ssh to VM and check if yggdrasil is active
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	status := false
	for i := 0; i < 5; i++ {
		status = tests.Wait(yggIP, "22")
		if status {
			break
		}
	}
	if status == false {
		t.Errorf("public ip not reachable")
	}

	verifyIPs := []string{publicIP, yggIP}
	tests.VerifyIPs("", verifyIPs)
	defer tests.DownWG()

	// ssh to VM by ygg_ip
	res, _ := tests.RemoteRun("root", yggIP, "printenv")
	assert.Contains(t, string(res), "TEST_VAR=this value for test")
}
