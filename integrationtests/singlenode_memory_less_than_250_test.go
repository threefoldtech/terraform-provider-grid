//go:build integration
// +build integration

package integrationtests

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestSingleNodeWithSmallMemDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./singlenode_with_memory_less_than_250M",
		Vars: map[string]interface{}{
			"public_key": pk,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)

	if err == nil {
		t.Errorf("Should fail with mem capacity can't be less that 250M but err is null")
	}

}
