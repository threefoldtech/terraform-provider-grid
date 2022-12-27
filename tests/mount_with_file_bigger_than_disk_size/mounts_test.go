package test

import (
	"log"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/tests"
)

func TestMountWithBiggerFileDeployment(t *testing.T) {
	/* Test case for deployeng a mount disk and try to create a file bigger than disk size.

	   **Test Scenario**

	   - Deploy a mount disk with size 1G.
	   - Check that the outputs not empty.
	   - Up wireguard.
	   - ssh to VM and try to create a file with size 1G.
	   - Destroy the deployment
	*/

	// retryable errors in terraform testing.
	// generate ssh keys for test
	pk, sk, err := tests.GenerateSSHKeyPair()
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

	terraform.InitAndApplyE(t, terraformOptions)

	publicIP := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, publicIP)

	// Up wireguard
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)
	_, err = tests.UpWg(wgConfig)
	assert.NoError(t, err)
	defer func() {
		_, err := tests.DownWG()
		assert.NoError(t, err)
	}()

	// ssh to VM and try to create a file with size 1G.
	pIP := strings.Split(publicIP, "/")[0]
	err = tests.Wait(pIP, "22")
	assert.NoError(t, err)
	res, err := tests.RemoteRun("root", pIP, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", sk)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "TEST_VAR=this value for test")

	_, err = tests.RemoteRun("root", pIP, "cd /app/ && dd if=/dev/vda bs=1G count=1 of=test.txt", sk)
	if err == nil {
		t.Errorf("should fail with out of memory")
	}
}
