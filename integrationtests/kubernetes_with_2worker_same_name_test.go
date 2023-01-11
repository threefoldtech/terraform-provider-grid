//go:build integration
// +build integration

package integrationtests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	tests "github.com/threefoldtech/terraform-provider-grid/integrationtests"
)

func TestKubernetesWith2WorkersSameNameDeployment(t *testing.T) {
	/* Test case for deployeng a k8s.

	   **Test Scenario**

	   - Deploy a k8s with 2 workers having the same name.
	   - Check that the deployment failed.

	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	publicKey, _, err := tests.GenerateSSHKeyPair()
	if err != nil {
		t.Fatal()
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./kubernetes_with_2worker_same_name",
		Vars: map[string]interface{}{
			"public_key": publicKey,
		},
		Parallelism: 1,
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)

	if err == nil {
		t.Errorf("k8s workers and master must have unique names")
	}

}
