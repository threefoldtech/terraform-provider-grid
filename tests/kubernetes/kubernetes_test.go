package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestKubernetesDeployment(t *testing.T) {
	// retryable errors in terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./",
	})

	terraform.InitAndApply(t, terraformOptions)
	defer terraform.Destroy(t, terraformOptions)

	// TODO: ping the master_public_ip
	master_public_ip := terraform.Output(t, terraformOptions, "master_public_ip")
	assert.NotEmpty(t, master_public_ip)

	// TODO: verify the wgConfig
	wgConfig := terraform.Output(t, terraformOptions, "wg_config")
	assert.NotEmpty(t, wgConfig)

}
