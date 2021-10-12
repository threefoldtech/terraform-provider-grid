default: build-dev

# Run acceptance tests
.PHONY: testacc build docs

build-dev:
	go get
	go mod tidy
	mkdir -p ~/.terraform.d/plugins/threefoldtechdev.com/providers/grid/0.2/linux_amd64/
	go build -o terraform-provider-grid
	mv terraform-provider-grid ~/.terraform.d/plugins/threefoldtechdev.com/providers/grid/0.2/linux_amd64/

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs	

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m


