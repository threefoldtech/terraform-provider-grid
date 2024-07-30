package integrationtests

import (
	"os"
	"strings"
	"testing"

	"github.com/go-redis/redis"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider/scheduler"
)

func TestZdbs(t *testing.T) {

	t.Run("zdb_test", func(t *testing.T) {
		/* Test case for deploying a zdb.

		   **Test Scenario**

		   - Deploy a zdb.
		   - Connect to the zdb.
		   - Write and read from the zdb.
		   - Assert that the written and read values match.
		   - Destroy the deployment

		*/
		if network, _ := os.LookupEnv("NETWORK"); network == "test" || network == "main" {
			t.Skip("https://github.com/threefoldtech/terraform-provider-grid/issues/770")
			return
		}

		password := "password123"
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./zdbs",
			Vars: map[string]interface{}{
				"password": password,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		if err != nil &&
			(strings.Contains(err.Error(), scheduler.NoNodesFoundErr.Error()) ||
				strings.Contains(err.Error(), "error creating threefold plugin client")) {
			t.Skip("couldn't find any available nodes")
			return
		}

		require.NoError(t, err)

		// Check that the outputs not empty
		deploymentID := terraform.Output(t, terraformOptions, "deployment_id")
		require.NotEmpty(t, deploymentID)

		zdbEndpoint := terraform.Output(t, terraformOptions, "zdb1_endpoint")
		require.NotEmpty(t, zdbEndpoint)

		zdbNamespace := terraform.Output(t, terraformOptions, "zdb1_namespace")
		require.NotEmpty(t, zdbNamespace)

		rdb := redis.NewClient(&redis.Options{
			Addr: zdbEndpoint,
		})
		_, err = rdb.Do("SELECT", zdbNamespace, password).Result()
		require.NoError(t, err)

		_, err = rdb.Set("key1", "val1", 0).Result()
		require.NoError(t, err)

		res, err := rdb.Get("key1").Result()
		require.NoError(t, err)
		require.Equal(t, res, "val1")
	})
}
