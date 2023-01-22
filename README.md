# Grid provider for terraform

A terraform provider for the [threefold grid](https://threefold.io) to manage your infrastructure using terraform.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.15
- [Gettting started document](https://library.threefold.me/info/manual/#/manual3_iac/grid3_terraform/manual__grid3_terraform_home)

- to use the built plugin in a terraform file, use the following provider config:

```terraform
terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
```

## Generating the docs

```bash
make docs
```

## Using the provider

```bash
cd examples/resources/singlenode
export MNEMONICS="<mnemonics words>"
export NETWORK="<network>" # dev or test
terraform init && terraform apply -parallelism=1 # creates resources defined in main.tf
terraform destroy -parallelism=1 # destroy the created resource
```

- For a tutorials, please visit the [wiki](https://library.threefold.me/info/manual/#/manual3_iac/grid3_terraform/manual__grid3_terraform_home) page.
- Detailed docs for resources and their arguments can be found in the [docs](docs).

## Building The Provider (for development only)

```bash
make
```

## Run tests

To run the tests, export MNEMONICS and NETWORK
export MNEMONICS="<mnemonics words>"
export NETWORK="<network>" # dev or test
run the following command

### running unit tests

```bash
make unittests
```

### running integration tests

```bash
make integrationtests
```

## Known Issues

- [parallelism=1](https://github.com/threefoldtech/terraform-provider-grid/issues/12).
- [increasing IPs in active deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/15).
- [introducing new nodes to kubernetes deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/13).
