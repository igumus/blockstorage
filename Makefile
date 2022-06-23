
.PHONY: all

all: help

## Build:
tidy: ## Tidy project
	go mod tidy -compat=1.17

clean: ## Cleans temporary folder
	rm -rf /tmp/peer*

build: clean tidy ## Builds project
	go build ./...

test: clean tidy ## Run unit tests
	go test -v

coverage: clean tidy ## Run code coverage
	go test -cover	

## Protoc:
gen: ## Generates go source files from protobuf.
	rm -v blockpb/*.pb.go
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
