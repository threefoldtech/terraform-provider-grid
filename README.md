# Grid provider for terraform
 - A resource, and a data source (`internal/provider/`),
 - Examples (`examples/`) 
## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15
-   A registered twin on the grid (make sure yggdrasil is running) [steps](https://github.com/threefoldtech/TFGRID/blob/development/wiki/tfgrid_substrate/substrate/grid_substrate_getting_started.md)
-   Redis running on localhost with port 6379

## Building The Provider (for development only)

Note: please clone all of the following repos in the same directory
- clone github.com/threefoldtech/zos  (switch to main branch)
- Clone github.com/threefoldtech/tf_terraform_provider (development branch)
- Enter the repository directory

```bash
go get
mkdir -p  ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
go build -o terraform-provider-grid 
mv terraform-provider-grid ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
```

- to use the built plugin in a terraform file, use the following provider config:
```
terraform {
  required_providers {
    grid = {
      source = "threefoldtech.com/providers/grid"
      version = "0.1"
    }
  }
}
```


## Using the provider
```bash
cd examples/resources/singlenode
export MNEMONICS="<mnemonics workds>"
export TWIN_ID="<your twin id>"
terraform init && terraform apply -parallelism=1
```
## Destroying deployment
```bash
terraform destroy -parallelism=1
```
## Current limitation

- [parallism=1](https://github.com/threefoldtech/terraform-provider-grid/issues/12)
- [windows  support](https://github.com/threefoldtech/terraform-provider-grid/issues/9)
- [increasing IPs in active deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/15)
- [introducing new nodes to kuberentes deployment](https://github.com/threefoldtech/terraform-provider-grid/issues/13)
- [multiple deployments on the same node](https://github.com/threefoldtech/terraform-provider-grid/issues/11)

## Troubleshooting

see [steps](https://github.com/threefoldtech/terraform-provider-grid/blob/development/TROUBLESHOOTING.md)