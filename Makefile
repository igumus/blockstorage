
.PHONY: all

all: help

## Build:
tidy: ## Tidy project
	go mod tidy -compat=1.17

build: ## Builds project
	go build ./...

test: ## Run unit tests
	rm -rf /tmp/peer
	go test -v

coverage: ## Run code coverage
	rm -rf /tmp/peer
	go test -cover	

## Protoc:
gen: ## Generates go source files from protobuf.
	rm -rf blockpb
	mkdir blockpb
	protoc --go_out=blockpb --go_opt=paths=source_relative --go-grpc_out=blockpb --go-grpc_opt=paths=source_relative -I api/protobuf store.proto

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
