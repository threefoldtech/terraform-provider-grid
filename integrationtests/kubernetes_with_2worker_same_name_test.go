//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestKubernetesWithNonExistNetworkDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./kubernetes_with_2worker_same_name",
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
