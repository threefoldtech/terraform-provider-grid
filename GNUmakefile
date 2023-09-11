GOPATH=$(shell go env GOPATH)

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

getverifiers:
	@echo "Installing staticcheck" && go get -u honnef.co/go/tools/cmd/staticcheck && go install honnef.co/go/tools/cmd/staticcheck
	@echo "Installing gocyclo" && go get -u github.com/fzipp/gocyclo/cmd/gocyclo && go install github.com/fzipp/gocyclo/cmd/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode && go install github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell && go install github.com/client9/misspell/cmd/misspell
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45

verifiers: fmt lint cyclo deadcode spelling staticcheck

checks: verifiers

fmt:
	@echo "Running $@"
	@gofmt -d .
	@terraform fmt -recursive

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golangci-lint run

cyclo:
	@echo "Running $@"
	@${GOPATH}/bin/gocyclo -over 100 .

deadcode:
	@echo "Running $@"
	@${GOPATH}/bin/deadcode -test $(shell go list ./...) || true

spelling:
	@echo "Running $@"
	@${GOPATH}/bin/misspell -i monitord -error `find .`

staticcheck:
	@echo "Running $@"
	@${GOPATH}/bin/staticcheck -- ./...

