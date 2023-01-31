# Using releases for different environments

- to use the `mainnet`'s version of the provider for `v1.7.0`, use the following configs:

  ```terraform
  terraform {
    required_providers {
      grid = {
        source = "threefoldtech/grid"
      }
    }
  }
  ```

- to use the `testnet`'s version of the provider for `v1.7.0`, use the following configs:

  ```terraform
  terraform{
    required_providers{
      grid = {
        source = "threeflodtech/grid"
        version = "v1.7.0-rcX"
      }
    }
  }
  ```

- to use the `devnet`'s version of the provider for `v1.7.0`, use the following configs:

  ```terraform
  terraform{
    required_providers{
      grid = {
        source = "threeflodtech/grid"
        version = "v1.7.0-dev"
      }
    }
  }
  ```

- to use the `qanet`'s version of the provider for `v1.7.0`, use the following configs:

  ```terraform
  terraform{
    required_providers{
      grid = {
        source = "threeflodtech/grid"
        version = "v1.7.0-qa"
      }
    }
  }
  ```
