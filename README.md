# Grid provider for terraform

A terraform provider for the [threefold grid](https://threefold.io) to manage your infrastructure using terraform.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.15
- [Gettting started document](https://library.threefold.me/info/manual/#/manual3_iac/grid3_terraform/manual__grid3_terraform_home)

## Using provider for different environments

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
        version = "1.7.0-rcX"
      }
    }
  }
  ```

- for devnet, qanet use `<VERSION>-dev` and `<VERSION>-qa` respectivly

## Generating the docs

```bash
make docs
```

## Using the provider

```bash
cd examples/resources/singlenode
export MNEMONICS="mnemonics words"
export NETWORK="network" # dev, qa, test, main
terraform init && terraform apply # creates resources defined in main.tf
terraform destroy # destroy the created resource
```

- For a tutorials, please visit the [wiki](https://library.threefold.me/info/manual/#/manual3_iac/grid3_terraform/manual__grid3_terraform_home) page.
- Detailed docs for resources and their arguments can be found in the [docs](docs).

## Building The Provider (for development only)

```bash
make
```

## Run tests

```bash
export MNEMONICS="mnemonics words"
export NETWORK="network" # dev, qa, test, main
```

- ### Unit tests

  ```bash
  make unittests
  ```

- ### Integration tests

  ```bash
  make integrationtests
  ```

## Known Issues

- [increasing IPs in active deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/15)

## Latest Releases

- Releasing for each environment is done using the methods in this [Wiki](wiki/release.md#releasing-for-each-environment)
- For latest releases [terraform-provider-grid](https://registry.terraform.io/providers/threefoldtech/grid/latest)

## Using example directory

- the examples directory contains some examples to show user how to use the provider so kindly note that
  - User should change the nodes to match the node that wants to deploy on
  - In examples that uses`SSH_KEY` default location is `file("~/.ssh/id_rsa.pub")` the path should be changed to match your public key location
