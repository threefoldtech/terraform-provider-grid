# Grid provider for terraform
 - A resource, and a data source (`internal/provider/`),
 - Examples (`examples/`) 
## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15

## Building The Provider

Note: please clone all of the following repos in the same directory
- clone github.com/threefoldtech/zos  (switch to master-3 branch)
- Clone github.com/threefoldtech/tf_terraform_provider (deployment_resource branch)
- Enter the repository directory
-
```bash
go get
mkdir -p  ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
go build -o terraform-provider-grid 
mv terraform-provider-grid ~/.terraform.d/plugins/threefoldtech.com/providers/grid/0.1/linux_amd64
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider
```bash
./msgbusd --twin <TWIN_ID> #run message bus with your twin id
cd examples/resources
export MNEMONICS="<mnemonics workds>"
export TWIN_ID="<your twin id>"
terraform init && terraform apply
```
## Destroying deployment
```bash
terraform destroy
```
