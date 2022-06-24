
.PHONY: all

all: help

## Build:
tidy: ## Tidy project
	go mod tidy -compat=1.17

clean: ## Cleans temporary folder
	rm -rf /tmp/peer*

build: clean tidy ## Builds project
	go build ./...

test-clean: ## Cleans test cache
	go clean -testcache

test: clean tidy test-clean ## Runs unit tests
	go test github.com/igumus/blockstorage{,/peer,/grpc}

coverage: clean tidy test-clean ## Run code coverage
	go test -cover github.com/igumus/blockstorage{,/peer,/grpc}

## Generations:
gen-proto: ## Generates go source files from protobuf.
	rm -fv blockpb/*.pb.go
	protoc --go_out=blockpb --go_opt=paths=source_relative --go-grpc_out=blockpb --go-grpc_opt=paths=source_relative -I api/protobuf store.proto

gen-mock: ## Generates mock objects
	go generate ./...

## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    %-20s%s\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  %s\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
