package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestSingleNodeWithZeroCPUDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Parallelism:  1,
	})

	_, err := terraform.InitAndApplyE(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	if err == nil {
		t.Errorf("Should fail with can't deploy with 0 cpu but err is null")
	}

}
