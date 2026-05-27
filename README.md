# Grid Terraform

A HashiCorp Terraform provider for managing decentralized cloud infrastructure through declarative configuration files. It enables infrastructure-as-code deployments of virtual machines, networks, Kubernetes clusters, and storage resources.

## What this is

This provider extends Terraform with resources for provisioning and managing workloads on a decentralized infrastructure grid. Users define infrastructure in `.tf` files and use standard Terraform workflows (`plan`, `apply`, `destroy`) to manage the lifecycle of their deployments. The provider handles node selection, resource allocation, network configuration, and contract management automatically.

## What this repository contains

- **Provider source code** — Go implementation of the Terraform plugin protocol
- **Resource implementations** — VM, network, Kubernetes, gateway, and storage resource types
- **Internal client libraries** — Grid client for node communication and contract management
- **Examples** — Sample configurations for single-node, multi-node, and advanced deployments
- **Documentation** — Provider docs and usage guides
- **Integration tests** — Terratest-based validation suite

## Role in the stack

This provider sits at the user-facing layer of the infrastructure stack. It translates Terraform declarations into grid operations by communicating with the chain layer for contract creation and with nodes for workload deployment. It is the primary DevOps automation tool for teams managing grid infrastructure at scale.

## Relation to ThreeFold

This technology is used within the ThreeFold ecosystem and was first deployed on the ThreeFold Grid. The component itself is designed as reusable infrastructure technology and should be understood by its technical function first, independent of any specific deployment.

## Ownership

This repository is owned and maintained by TF-Tech NV, a Belgian company responsible for the development and maintenance of this technology.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.25

## Using the provider

For the latest release:

```terraform
terraform {
  required_providers {
    grid = {
      source = "threefoldtech/grid"
    }
  }
}
```

For testnet, use a release candidate version:

```terraform
terraform {
  required_providers {
    grid = {
      source  = "threefoldtech/grid"
      version = "1.7.0-rcX"
    }
  }
}
```

For devnet or qanet, append `-dev` or `-qa` to the version respectively.

### Quick start

```bash
cd examples/resources/singlenode
export MNEMONIC="mnemonic words"
export NETWORK="network" # dev, qa, test, main
terraform init && terraform apply
terraform destroy
```

- For tutorials, visit the [wiki](https://manual.grid.tf/documentation/system_administrators/terraform/terraform_readme.html#get-started).
- Detailed docs for resources and their arguments can be found in the [docs](docs).

## Building the provider (for development)

```bash
make
```

## Generating the docs

```bash
make docs
```

## Run tests

```bash
export MNEMONIC="mnemonic words"
export NETWORK="network" # dev, qa, test, main
```

### Unit tests

```bash
make unittests
```

### Integration tests

```bash
make integration
```

To run a single test:

```bash
cd integrationtests
go test . -run <TestNameFunction> -v --tags=integration
go test . -run <TestNameFunction/SubFunctionName> -v --tags=integration
```

## Known issues

- [increasing IPs in active deployment](https://github.com/threefoldtech/grid_terraform/issues/15)
- [same private IPs for parallel deployments](https://github.com/threefoldtech/grid_terraform/issues/781#issuecomment-1865961184)

## Latest releases

- Releasing for each environment is documented in the [release wiki](wiki/release.md#releasing-for-each-environment).
- For the latest releases, see the [Terraform Registry](https://registry.terraform.io/providers/threefoldtech/grid/latest).

## Using the examples directory

The `examples` directory contains sample configurations. When using them:

- Change the nodes to match the nodes you want to deploy on.
- In examples that use `SSH_KEY`, the default location is `file("~/.ssh/id_rsa.pub")`. Update the path to match your public key location.

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.
