//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestSingleNodeWithZeroCPUDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Log(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./singlevm_with_zero_cpu",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)

	if err == nil {
		t.Errorf("Should fail with can't deploy with 0 cpu but err is null")
	}

}
