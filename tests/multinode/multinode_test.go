package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestMultiNodeDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
	})

	terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	// TODO: ping the public ip
	public_ip := terraform.Output(t, terraformOptions, "public_ip")
	assert.NotEmpty(t, public_ip)

	// TODO: ping the container
	node1_container2_ip := terraform.Output(t, terraformOptions, "node1_container2_ip")
	assert.NotEmpty(t, node1_container2_ip)

	// TODO: ping the container
	node1_container1_ip := terraform.Output(t, terraformOptions, "node1_container1_ip")
	assert.NotEmpty(t, node1_container1_ip)

	// TODO: verify the wgConfig
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

}
