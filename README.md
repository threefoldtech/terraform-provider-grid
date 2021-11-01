# Grid provider for terraform
 - A resource, and a data source (`internal/provider/`),
 - Examples (`examples/`) 
## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15
-   A registered twin on the grid with a funed account [steps](https://library.threefold.me/info/threefold#/manual_tfgrid3/threefold__grid3_get_started)

- Only if not using the rmb proxy (enabled by default): Redis running on localhost with port 6379 and yggdrasil

## Building The Provider (for development only)

```bash
make
```

- to use the built plugin in a terraform file, use the following provider config:
```
terraform {
  required_providers {
    grid = {
      source = "threefoldtech.com/providers/grid"
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
terraform init && terraform apply -parallelism=1
```
## Destroying deployment
```bash
terraform destroy -parallelism=1
```
## Current limitation

- [parallism=1](https://github.com/threefoldtech/terraform-provider-grid/issues/12)
- [increasing IPs in active deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/15)
- [introducing new nodes to kuberentes deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/13)
- [multiple deployments on the same node](https://github.com/threefoldtech/terraform-provider-grid/issues/11)

## Troubleshooting

see [steps](https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)
