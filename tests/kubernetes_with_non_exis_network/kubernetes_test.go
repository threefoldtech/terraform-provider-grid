package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestKubernetesWithNonExistNetworkDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
	})

	_, err := terraform.InitAndApplyE(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	if err == nil {
		t.Errorf("The deployment should fail but err is null")
	}

}
