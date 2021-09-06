package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestSingleNodeWithSmallMemDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
	})

	_, err := terraform.InitAndApplyE(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	if err == nil {
		t.Errorf("Should fail with mem capacity can't be less that 250M but err is null")
	}

}
