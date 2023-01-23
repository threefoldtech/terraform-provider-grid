package integrationtests

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestGateWay(t *testing.T) {
	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		log.Fatalf("failed to generate ssh key pair: %s", err.Error())
	}

	t.Run("gateway_name", func(t *testing.T) {
		/* Test case for deployeng a gateway name proxy.

		   **Test Scenario**

		   - Deploy a gateway name.
		   - Deploy a vm.
		   - Assert deployments outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_name",
			Vars: map[string]interface{}{
				"public_key": publicKey,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		fqdn := terraform.Output(t, terraformOptions, "fqdn")
		assert.NotEmpty(t, fqdn)

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		ok := TestConnection(yggIP, "22")
		assert.True(t, ok)

		_, err = RemoteRun("root", yggIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("http://%s", fqdn))
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		defer response.Body.Close()
		assert.NoError(t, err)
		assert.Contains(t, string(body), "Directory listing for")

	})

	t.Run("gateway_fqdn", func(t *testing.T) {
		/* Test case for deployeng a gateway with fdqn.

		   **Test Scenario**

		   - Deploy a gateway with fdqn.
		   - Deploy a vm.
		   - Assert that outputs are not empty.
		   - Run python server on vm.
		   - Make an http request to fqdn and assert that the response is correct.
		   - Destroy the deployment
		*/

		fqdn := "hamada1.3x0.me" // points to node 15 devnet

		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./gateway_with_fqdn",
			Vars: map[string]interface{}{
				"public_key": publicKey,
				"fqdn":       fqdn,
			},
			Parallelism: 1,
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		fqdn = terraform.Output(t, terraformOptions, "fqdn")
		assert.NotEmpty(t, fqdn)

		yggIP := terraform.Output(t, terraformOptions, "ygg_ip")
		assert.NotEmpty(t, yggIP)

		ok := TestConnection(yggIP, "22")
		assert.True(t, ok)

		_, err = RemoteRun("root", yggIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		response, err := http.Get(fmt.Sprintf("http://%s", fqdn))
		assert.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		defer response.Body.Close()
		assert.NoError(t, err)
		assert.Contains(t, string(body), "Directory listing for")
	})
}
