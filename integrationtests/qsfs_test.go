//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"os/exec"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestMultiNodeDeployment(t *testing.T) {
	/* Test case for deployeng a QSFS check metrics.
	   **Test Scenario**
	   - Deploy a qsfs.
	   - Check that the outputs not empty.
	   - Check that containers is reachable
	   - write to a file. number of syscalls for write should increase try open, read, create, rename or etc number of syscalls should increase
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, sk, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./qsfs_check_metrics",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	// Check that the outputs not empty
	metrics := terraform.Output(t, terraformOptions, "metrics")
	assert.NotEmpty(t, metrics)

	ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, ygg_ip)

	// err = tests.Wait(ygg_ip, "22")
	// assert.NoError(t, err)

	// isIPReachable := []string{ygg_ip, metrics}
	// err = tests.isIPReachable("", isIPReachable, sk)
	// assert.NoError(t, err)

	// get metrics
	cmd := exec.Command("curl", metrics)
	output, _ := cmd.Output()

	// try write to a file in mounted disk
	_, err = tests.RemoteRun("root", ygg_ip, "cd /qsfs && echo test >> test", sk)
	assert.NoError(t, err)
	// get metrics after write
	cmd2 := exec.Command("curl", metrics)
	output_after_write, _ := cmd2.Output()

	// check that syscalls for write should increase
	assert.NotEqual(t, output, output_after_write)
}
