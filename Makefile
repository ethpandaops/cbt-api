.PHONY: help install-tools generate build run clean fmt lint test unit-test integration-test export-test-data

# Colors for output (use printf for cross-platform compatibility)
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
RESET := \033[0m

# xatu-cbt repository configuration
XATU_CBT_REPO := https://github.com/ethpandaops/xatu-cbt.git
XATU_CBT_VERSION ?= master
XATU_CBT_DIR := ./.xatu-cbt

# Paths
PROTO_PATH := $(XATU_CBT_DIR)/pkg/proto/clickhouse
TMP_DIR := ./tmp
OUTPUT_FILE := ./openapi.yaml
PREPROCESS_TOOL := ./cmd/tools/openapi-preprocess
POSTPROCESS_TOOL := ./cmd/tools/openapi-postprocess

# Test data export configuration (for exporting from production ClickHouse)
TESTDATA_EXPORT_HOST ?= http://localhost:8123
TESTDATA_EXPORT_DATABASE ?= mainnet
TESTDATA_DIR := internal/integrationtest/testdata

# Get googleapis path
GOOGLEAPIS_PATH := $(shell go list -m -f '{{.Dir}}' github.com/googleapis/googleapis 2>/dev/null || echo "")

.DEFAULT_GOAL := help

help: ## Show this help message
	@printf "$(CYAN)xatu-cbt-api Makefile$(RESET)\n"
	@printf "\n"
	@printf "$(GREEN)Main workflow:$(RESET)\n"
	@printf "  $(CYAN)make install-tools$(RESET)  # One-time setup: install required dependencies\n"
	@printf "  $(CYAN)make generate$(RESET)       # Generate OpenAPI spec and server code\n"
	@printf "  $(CYAN)make build$(RESET)          # Build the API server binary\n"
	@printf "  $(CYAN)make run$(RESET)            # Run the API server\n"
	@printf "\n"
	@printf "$(GREEN)Development:$(RESET)\n"
	@printf "  $(CYAN)make clean$(RESET)          # Remove generated files and build artifacts\n"
	@printf "  $(CYAN)make fmt$(RESET)            # Format Go code\n"
	@printf "  $(CYAN)make lint$(RESET)           # Run linters\n"
	@printf "  $(CYAN)make test$(RESET)           # Run all tests (unit + integration)\n"
	@printf "\n"
	@printf "$(GREEN)Testing:$(RESET)\n"
	@printf "  $(CYAN)make unit-test$(RESET)      # Run unit tests only\n"
	@printf "  $(CYAN)make integration-test$(RESET) # Run integration tests\n"
	@printf "  $(CYAN)make export-test-data$(RESET) # Export test data from production ClickHouse\n"

# Install required development tools (one-time setup)
install-tools:
	@printf "$(CYAN)==> Installing required tools...$(RESET)\n"
	@if ! command -v protoc >/dev/null 2>&1; then \
		printf "$(YELLOW)==> Installing protoc...$(RESET)\n"; \
		if [ "$$(uname)" = "Linux" ]; then \
			if command -v apk >/dev/null 2>&1; then \
				apk add --no-cache protobuf-dev protoc; \
			elif command -v apt-get >/dev/null 2>&1; then \
				if [ "$$(id -u)" -eq 0 ]; then \
					apt-get update && apt-get install -y protobuf-compiler; \
				else \
					sudo apt-get update && sudo apt-get install -y protobuf-compiler; \
				fi; \
			else \
				printf "$(RED)Error: No supported package manager found$(RESET)\n"; \
				exit 1; \
			fi; \
		elif [ "$$(uname)" = "Darwin" ]; then \
			brew install protobuf; \
		else \
			printf "$(RED)Error: Unsupported OS. Please install protoc manually from https://github.com/protocolbuffers/protobuf/releases$(RESET)\n"; \
			exit 1; \
		fi; \
	else \
		printf "$(GREEN)✓ protoc already installed$(RESET)\n"; \
	fi
	@go install github.com/kollalabs/protoc-gen-openapi@latest
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go get github.com/getkin/kin-openapi/openapi3@latest
	@go get github.com/googleapis/googleapis@latest
	@go get gopkg.in/yaml.v3@latest
	@printf "$(GREEN)✓ Tools installed$(RESET)\n"

# Generate all code (OpenAPI spec + server implementation)
generate: .clone-xatu-cbt .build-tools .openapi .generate-descriptors .generate-server
	@printf "$(GREEN)✓ Code generation complete$(RESET)\n"

# Version information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
DIRTY_SUFFIX := $(shell git diff --quiet || echo "-dirty")

# Build ldflags
LDFLAGS := -s -w \
	-X github.com/ethpandaops/xatu-cbt-api/internal/version.Release=$(VERSION)-$(GIT_COMMIT)$(DIRTY_SUFFIX) \
	-X github.com/ethpandaops/xatu-cbt-api/internal/version.GitCommit=$(GIT_COMMIT)

# Build the API server binary
build:
	@printf "$(CYAN)==> Building API server...$(RESET)\n"
	@printf "$(CYAN)    Version: $(VERSION)-$(GIT_COMMIT)$(DIRTY_SUFFIX)$(RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server
	@printf "$(GREEN)✓ Server built: bin/server$(RESET)\n"

# Run the API server
run: build
	@printf "$(CYAN)==> Starting API server...$(RESET)\n"
	@./bin/server

# Internal targets (not meant to be called directly)
.clone-xatu-cbt:
	@if [ -d "$(XATU_CBT_DIR)" ]; then \
		printf "$(YELLOW)==> Updating xatu-cbt repository...$(RESET)\n"; \
		cd $(XATU_CBT_DIR) && git fetch origin && git checkout $(XATU_CBT_VERSION) 2>/dev/null || true; \
	else \
		printf "$(CYAN)==> Cloning xatu-cbt repository (commit: $(XATU_CBT_VERSION))...$(RESET)\n"; \
		git clone $(XATU_CBT_REPO) $(XATU_CBT_DIR) > /dev/null 2>&1 && cd $(XATU_CBT_DIR) && git checkout $(XATU_CBT_VERSION) > /dev/null 2>&1; \
	fi
	@printf "$(GREEN)✓ xatu-cbt repository ready at $(XATU_CBT_VERSION)$(RESET)\n"

.build-tools:
	@printf "$(CYAN)==> Building code generation tools...$(RESET)\n"
	@go build -o bin/openapi-preprocess $(PREPROCESS_TOOL)
	@printf "$(GREEN)✓ Built: bin/openapi-preprocess$(RESET)\n"

.openapi: .build-tools .clone-xatu-cbt .generate-descriptors
	@printf "$(CYAN)==> Generating OpenAPI 3.0 from annotated protos...$(RESET)\n"
	@mkdir -p $(TMP_DIR)
	@if [ -z "$(GOOGLEAPIS_PATH)" ]; then \
		printf "$(RED)Error: googleapis not found. Installing...$(RESET)\n"; \
		go get github.com/googleapis/googleapis@latest; \
	fi
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	protoc \
		--openapi_out=$(TMP_DIR) \
		--openapi_opt=naming=proto \
		--proto_path=$(PROTO_PATH) \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(PROTO_PATH)/*.proto
	@printf "$(GREEN)✓ OpenAPI spec generated: $(OUTPUT_FILE)$(RESET)\n"
	@printf "$(CYAN)==> Pre-processing OpenAPI spec...$(RESET)\n"
	@go run $(PREPROCESS_TOOL) \
		--input $(TMP_DIR)/openapi.yaml \
		--output $(OUTPUT_FILE) \
		--proto-path $(PROTO_PATH) \
		--descriptor .descriptors.pb
	@printf "$(GREEN)✓ Pre-processed spec generated: $(OUTPUT_FILE)$(RESET)\n"

.generate-descriptors: .clone-xatu-cbt
	@printf "$(CYAN)==> Generating protobuf descriptors...$(RESET)\n"
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	protoc \
		--descriptor_set_out=.descriptors.pb \
		--include_imports \
		--proto_path=$(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(XATU_CBT_DIR)/pkg/proto/clickhouse/*.proto \
		$(XATU_CBT_DIR)/pkg/proto/clickhouse/clickhouse/*.proto
	@printf "$(GREEN)✓ Protobuf descriptors generated: .descriptors.pb$(RESET)\n"

.generate-server: .openapi .generate-descriptors
	@printf "$(CYAN)==> Generating server interface from OpenAPI spec...$(RESET)\n"
	@mkdir -p internal/handlers
	@oapi-codegen --config oapi-codegen.yaml openapi.yaml > internal/handlers/generated.go
	@printf "$(CYAN)==> Post-processing generated code...$(RESET)\n"
	@go run $(POSTPROCESS_TOOL) --input internal/handlers/generated.go
	@printf "$(GREEN)✓ Server interface generated: internal/handlers/generated.go$(RESET)\n"
	@printf "$(CYAN)==> Generating server implementation...$(RESET)\n"
	@mkdir -p internal/server
	@go run ./cmd/tools/generate-implementation \
		--openapi openapi.yaml \
		--proto-path $(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--output internal/server/implementation.go
	@printf "$(CYAN)==> Copying OpenAPI spec for embedding...$(RESET)\n"
	@cp openapi.yaml internal/server/openapi.yaml
	@printf "$(GREEN)✓ Server implementation generated: internal/server/implementation.go$(RESET)\n"

# Clean generated files and build artifacts
clean:
	@printf "$(CYAN)==> Cleaning generated files...$(RESET)\n"
	@rm -rf $(TMP_DIR)
	@rm -f $(OUTPUT_FILE)
	@rm -f .descriptors.pb
	@rm -rf bin/
	@rm -f internal/handlers/generated.go
	@rm -f internal/server/implementation.go
	@rm -f internal/server/openapi.yaml
	@rm -rf $(XATU_CBT_DIR)
	@printf "$(GREEN)✓ Cleaned$(RESET)\n"

# Format Go code
fmt:
	@printf "$(CYAN)==> Formatting Go code...$(RESET)\n"
	@go fmt ./...
	@printf "$(GREEN)✓ Formatted$(RESET)\n"

# Run linters
lint:
	@printf "$(CYAN)==> Running linters...$(RESET)\n"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		printf "$(YELLOW)golangci-lint not installed, skipping...$(RESET)\n"; \
	fi

# Export test data from production ClickHouse
# Auto-detects tables from openapi.yaml (any table exposed via API)
export-test-data:
	@printf "$(CYAN)==> Exporting test data from $(TESTDATA_EXPORT_HOST) database $(TESTDATA_EXPORT_DATABASE)...$(RESET)\n"
	@mkdir -p $(TESTDATA_DIR)
	@printf "$(CYAN)==> Auto-detecting tables from openapi.yaml...$(RESET)\n"
	@for table in $$(grep -oE '/api/v1/[a-z_0-9]+' openapi.yaml | sed 's|/api/v1/||' | sort -u); do \
		printf "$(CYAN)  -> Exporting $$table...$(RESET)\n"; \
		curl -sS "$(TESTDATA_EXPORT_HOST)" \
			--data-binary "SELECT * FROM $(TESTDATA_EXPORT_DATABASE).$$table FINAL LIMIT 2 FORMAT JSON" \
			-o "$(TESTDATA_DIR)/$$table.json" || printf "$(YELLOW)  ⚠️  Failed to export $$table (may be empty or inaccessible)$(RESET)\n"; \
	done
	@printf "$(GREEN)✓ Test data exported to $(TESTDATA_DIR)/$(RESET)\n"

# Run all tests (unit + integration)
test: unit-test integration-test
	@printf "$(GREEN)✓ All tests passed$(RESET)\n"

# Run unit tests only (excludes integration tests)
unit-test:
	@printf "$(CYAN)==> Running unit tests...$(RESET)\n"
	@go install gotest.tools/gotestsum@latest
	@gotestsum --raw-command go test -v -race -failfast -coverprofile=coverage.out -covermode=atomic -json $$(go list ./... | grep -v /integrationtest) && \
		printf "$(GREEN)✓ Unit tests passed$(RESET)\n"

# Run integration tests
integration-test:
	@printf "$(CYAN)==> Running integration tests...$(RESET)\n"
	@bash -c "set -o pipefail; go test -v -race -timeout=5m ./internal/integrationtest/... | tee integration-test.log" && \
		printf "$(GREEN)✓ Integration tests passed$(RESET)\n"
