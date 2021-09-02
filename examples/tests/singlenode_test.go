package test

import (
        "testing"

        "github.com/gruntwork-io/terratest/modules/terraform"
        "github.com/stretchr/testify/assert"
)

func TestSingleNodeDeployment(t *testing.T) {
	t.Parallel()
        // retryable errors in terraform testing.
        terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
                TerraformDir: "../resources/singlenode/",

        })


        terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)
        public_ip := terraform.Output(t, terraformOptions, "public_ip")
        assert.NotEmpty(t,public_ip)

	node1_container2_ip := terraform.Output(t, terraformOptions, "node1_container2_ip")
        assert.NotEmpty(t,node1_container2_ip)

	node1_container1_ip := terraform.Output(t, terraformOptions, "node1_container1_ip")
        assert.NotEmpty(t,node1_container1_ip)

	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
        assert.NotEmpty(t,wgConfig)


}

