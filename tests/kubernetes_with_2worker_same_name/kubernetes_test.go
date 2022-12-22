package test

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestKubernetesWithNonExistNetworkDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.SshKeys()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)

	if err == nil {
		t.Errorf("k8s workers and master must have unique names")
	}

}
