package integrationtests

import (
	"testing"

	"github.com/go-redis/redis"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestZdbs(t *testing.T) {

	t.Run("zdb_test", func(t *testing.T) {
		/* Test case for deployeng a singlenode.

		   **Test Scenario**

		   - Deploy a zdb.
		   - Connect to the zdb.
		   - Write and read from the zdb.
		   - Assert that the written and read values match.
		   - Destroy the deployment

		*/
		password := "password123"
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: "./zdbs",
			Parallelism:  1,
			Vars: map[string]interface{}{
				"password": password,
			},
		})
		defer terraform.Destroy(t, terraformOptions)

		_, err := terraform.InitAndApplyE(t, terraformOptions)
		assert.NoError(t, err)

		// Check that the outputs not empty
		deploymentID := terraform.Output(t, terraformOptions, "deployment_id")
		assert.NotEmpty(t, deploymentID)

		zdbEndpoint := terraform.Output(t, terraformOptions, "zdb1_endpoint")
		assert.NotEmpty(t, zdbEndpoint)

		zdbNamespace := terraform.Output(t, terraformOptions, "zdb1_namespace")
		assert.NotEmpty(t, zdbNamespace)

		rdb := redis.NewClient(&redis.Options{
			Addr: zdbEndpoint,
		})
		_, err = rdb.Do("SELECT", zdbNamespace, password).Result()
		assert.NoError(t, err)

		_, err = rdb.Set("key1", "val1", 0).Result()
		assert.NoError(t, err)

		res, err := rdb.Get("key1").Result()
		assert.NoError(t, err)
		assert.Equal(t, res, "val1")
	})
}
