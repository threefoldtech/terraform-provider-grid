package integrationtests

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestGateWayPrivate(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("gateway_name_private", func(t *testing.T) {
		/* Test case for deploying a gateway name proxy.

		   **Test Scenario**

		   - Deploy a vm.
		   - Deploy a gateway name on the vm private network.
		   - Assert deployments outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_name_private",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		fqdn := terraform.Output(t, terraformOptions, "fqdn")
		assert.NotEmpty(t, fqdn)

		ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, ygg_ip)

		ok := TestConnection(ygg_ip, "22")
		assert.True(t, ok)

		_, err = RemoteRun("root", ygg_ip, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("http://%s", fqdn))
		assert.NoError(t, err)

		body, err := io.ReadAll(response.Body)
		if body != nil {
			defer response.Body.Close()
		}
		assert.NoError(t, err)
		assert.Contains(t, string(body), "Directory listing for")

	})

	t.Run("gateway_fqdn_private", func(t *testing.T) {
		t.SkipNow()
		/* Test case for deploying a gateway with fdqn.

		   **Test Scenario**

		   - Deploy a vm.
		   - Deploy a gateway with fdqn on the vm private network.
		   - Assert that outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		fqdn := "hamada1.3x0.me" // points to node 11 devnet

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_with_fqdn_private",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"fqdn":       fqdn,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		fqdn = terraform.Output(t, terraformOptions, "fqdn")
		assert.NotEmpty(t, fqdn)

		ygg_ip := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, ygg_ip)

		ok := TestConnection(ygg_ip, "22")
		assert.True(t, ok)

		_, err = RemoteRun("root", ygg_ip, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("http://%s", fqdn))
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		if body != nil {
			defer response.Body.Close()
		}
		assert.NoError(t, err)
		assert.Contains(t, string(body), "Directory listing for")
	})
}
