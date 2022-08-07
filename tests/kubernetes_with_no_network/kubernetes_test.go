//go:build integration
// +build integration

package test

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestKubernetesWithNonExistNetworkDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	// generate ssh keys for test
	tests.SshKeys()
	publicKey := os.Getenv("PUBLICKEY")
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err := terraform.InitAndApplyE(t, terraformOptions)

	if err == nil {
		t.Errorf("The deployment should fail but err is null")
	}

}
