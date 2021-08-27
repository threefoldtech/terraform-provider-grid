# Grid provider for terraform
 - A resource, and a data source (`internal/provider/`),
 - Examples (`examples/`) 
## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15
-   A registered twin on the grid (make sure yggdrasil is running) [steps](https://github.com/threefoldtech/TFGRID/blob/development/wiki/tfgrid_substrate/substrate/grid_substrate_getting_started.md)

## Building The Provider

Note: please clone all of the following repos in the same directory
- clone github.com/threefoldtech/zos  (switch to master-3 branch)
- Clone github.com/threefoldtech/tf_terraform_provider (deployment_resource branch)
- Enter the repository directory

```bash
go get
mkdir -p  ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
go build -o terraform-provider-grid 
mv terraform-provider-grid ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
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
