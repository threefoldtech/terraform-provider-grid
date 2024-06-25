package integrationtests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestPresearch(t *testing.T) {
	/* Test case for deploying a presearch.

	   **Test Scenario**

	   - Deploy a presearch.
	   - Check that the outputs not empty.
	   - Check that node is reachable.
	   - Check that vm is reachable
	   - Check that env variables set successfully
	   - Check prenode service is running
	   - Destroy the deployment
	*/
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	presearchRegistrationCode := "e5083a8d0a6362c6cf7a3078bfac81e3"
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./presearch",
		Vars: map[string]interface{}{
			"public_key":                  publicKey,
			"presearch_registration_code": presearchRegistrationCode,
		},
	})
	defer terraform.Destroy(t, terraformOptions)

	_, err = terraform.InitAndApplyE(t, terraformOptions)
	if err != nil && errors.As(err, &retry.FatalError{Underlying: scheduler.NoNodesFoundErr}) {
		t.Skip("couldn't find any available nodes")
		return
	}

	assert.NoError(t, err)

	yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
	assert.NotEmpty(t, yggIP)

	ok := TestConnection(yggIP, "22")
	assert.True(t, ok)

	output, err := RemoteRun("root", yggIP, "cat /proc/1/environ", privateKey)
	assert.NoError(t, err)
	assert.Contains(t, string(output), fmt.Sprintf("PRESEARCH_REGISTRATION_CODE=%s", presearchRegistrationCode))

	ticker := time.NewTicker(2 * time.Second)
	for now := time.Now(); time.Since(now) < 1*time.Minute; {
		<-ticker.C
		output, err = RemoteRun("root", yggIP, "zinit list", privateKey)
		if err == nil && strings.Contains(output, "prenode: Success") {
			break
		}
	}

	assert.NoError(t, err)
	assert.Contains(t, output, "prenode: Success")
}
