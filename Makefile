##@ General

.PHONY: all
all: help

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: vendor
vendor: ## Update the vendor directory
	go mod tidy
	go mod vendor

.PHONY: generate
generate: install-tools generate-swagger ## Run all code generations
	go generate ./...

generate-swagger: install-tools ## Run swagger code generation
	swag init -g server/ -g swagger.go --outputTypes go -output server/docs
	go generate swagger.go

format-swagger: install-tools ## Format swagger godoc's
	swag fmt -g server/

install-tools: ## Install development tools
	hash swag > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi ;

##@ Test

.PHONY: test
test: ## Run all tests
	go test -race -cover -coverprofile coverage.out -timeout 180s  github.com/discuitnet/discuit/...