DIRS := . $(shell find integrationtests examples -type d)
GARBAGE_PATTERNS := terraform.tfstate.backup terraform.tfstate .terraform.lock.hcl state.json .terraform
GARBAGE := $(foreach DIR,$(DIRS),$(addprefix $(DIR)/,$(GARBAGE_PATTERNS)))

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

unittests:
	go test -v `go list ./... | grep -v integrationtests`

integration: clean build-dev
	go test -v ./integrationtests/... --tags=integration -timeout 1800s

tests: unittests integrationtests

clean:
	rm -rf $(GARBAGE)

lint:
	@echo "Running $@"
	golangci-lint run -c .golangci.yml --timeout 10m

get_linter:
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45
