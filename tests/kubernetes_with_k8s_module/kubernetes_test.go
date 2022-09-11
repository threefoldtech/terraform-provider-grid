package test

import (
	"testing"

    "github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestKubernetesDeployment(t *testing.T) {
    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: "./",
        Parallelism: 1,
    })
    defer terraform.Destroy(t, terraformOptions)

    terraform.InitAndApply(t, terraformOptions)
    out := terraform.Output(t, terraformOptions, "master_yggip")
    assert.NotEmpty(t, out)

    out = terraform.Output(t, terraformOptions, "w0_yggip")
    assert.NotEmpty(t, out)

    out = terraform.Output(t, terraformOptions, "w1_yggip")
    assert.NotEmpty(t, out)
}
