.PHONY: help install-tools generate build run clean fmt lint test unit-test integration-test export-test-data

# Colors for output
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
	@echo "$(CYAN)xatu-cbt-api Makefile$(RESET)"
	@echo ""
	@echo "$(GREEN)Main workflow:$(RESET)"
	@echo "  $(CYAN)make install-tools$(RESET)  # One-time setup: install required dependencies"
	@echo "  $(CYAN)make generate$(RESET)       # Generate OpenAPI spec and server code"
	@echo "  $(CYAN)make build$(RESET)          # Build the API server binary"
	@echo "  $(CYAN)make run$(RESET)            # Run the API server"
	@echo ""
	@echo "$(GREEN)Development:$(RESET)"
	@echo "  $(CYAN)make clean$(RESET)          # Remove generated files and build artifacts"
	@echo "  $(CYAN)make fmt$(RESET)            # Format Go code"
	@echo "  $(CYAN)make lint$(RESET)           # Run linters"
	@echo "  $(CYAN)make test$(RESET)           # Run all tests (unit + integration)"
	@echo ""
	@echo "$(GREEN)Testing:$(RESET)"
	@echo "  $(CYAN)make unit-test$(RESET)      # Run unit tests only"
	@echo "  $(CYAN)make integration-test$(RESET) # Run integration tests"
	@echo "  $(CYAN)make export-test-data$(RESET) # Export test data from production ClickHouse"

# Install required development tools (one-time setup)
install-tools:
	@echo "$(CYAN)==> Installing required tools...$(RESET)"
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "$(YELLOW)==> Installing protoc...$(RESET)"; \
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
				echo "$(RED)Error: No supported package manager found$(RESET)"; \
				exit 1; \
			fi; \
		elif [ "$$(uname)" = "Darwin" ]; then \
			brew install protobuf; \
		else \
			echo "$(RED)Error: Unsupported OS. Please install protoc manually from https://github.com/protocolbuffers/protobuf/releases$(RESET)"; \
			exit 1; \
		fi; \
	else \
		echo "$(GREEN)✓ protoc already installed$(RESET)"; \
	fi
	@go install github.com/kollalabs/protoc-gen-openapi@latest
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go get github.com/getkin/kin-openapi/openapi3@latest
	@go get github.com/googleapis/googleapis@latest
	@go get gopkg.in/yaml.v3@latest
	@echo "$(GREEN)✓ Tools installed$(RESET)"

# Generate all code (OpenAPI spec + server implementation)
generate: .clone-xatu-cbt .build-tools .openapi .generate-descriptors .generate-server
	@echo "$(GREEN)✓ Code generation complete$(RESET)"

# Build the API server binary
build:
	@echo "$(CYAN)==> Building API server...$(RESET)"
	@go build -o bin/server ./cmd/server
	@echo "$(GREEN)✓ Server built: bin/server$(RESET)"

# Run the API server
run: build
	@echo "$(CYAN)==> Starting API server...$(RESET)"
	@./bin/server

# Internal targets (not meant to be called directly)
.clone-xatu-cbt:
	@if [ -d "$(XATU_CBT_DIR)" ]; then \
		echo "$(YELLOW)==> Updating xatu-cbt repository...$(RESET)"; \
		cd $(XATU_CBT_DIR) && git fetch origin && git checkout $(XATU_CBT_VERSION) 2>/dev/null || true; \
	else \
		echo "$(CYAN)==> Cloning xatu-cbt repository (commit: $(XATU_CBT_VERSION))...$(RESET)"; \
		git clone $(XATU_CBT_REPO) $(XATU_CBT_DIR) > /dev/null 2>&1 && cd $(XATU_CBT_DIR) && git checkout $(XATU_CBT_VERSION) > /dev/null 2>&1; \
	fi
	@echo "$(GREEN)✓ xatu-cbt repository ready at $(XATU_CBT_VERSION)$(RESET)"

.build-tools:
	@echo "$(CYAN)==> Building code generation tools...$(RESET)"
	@go build -o bin/openapi-preprocess $(PREPROCESS_TOOL)
	@echo "$(GREEN)✓ Built: bin/openapi-preprocess$(RESET)"

.openapi: .build-tools .clone-xatu-cbt .generate-descriptors
	@echo "$(CYAN)==> Generating OpenAPI 3.0 from annotated protos...$(RESET)"
	@mkdir -p $(TMP_DIR)
	@if [ -z "$(GOOGLEAPIS_PATH)" ]; then \
		echo "$(RED)Error: googleapis not found. Installing...$(RESET)"; \
		go get github.com/googleapis/googleapis@latest; \
	fi
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	protoc \
		--openapi_out=$(TMP_DIR) \
		--openapi_opt=naming=proto \
		--proto_path=$(PROTO_PATH) \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(PROTO_PATH)/*.proto
	@echo "$(GREEN)✓ OpenAPI spec generated: $(OUTPUT_FILE)$(RESET)"
	@echo "$(CYAN)==> Pre-processing OpenAPI spec...$(RESET)"
	@go run $(PREPROCESS_TOOL) \
		--input $(TMP_DIR)/openapi.yaml \
		--output $(OUTPUT_FILE) \
		--proto-path $(PROTO_PATH) \
		--descriptor .descriptors.pb
	@echo "$(GREEN)✓ Pre-processed spec generated: $(OUTPUT_FILE)$(RESET)"

.generate-descriptors: .clone-xatu-cbt
	@echo "$(CYAN)==> Generating protobuf descriptors...$(RESET)"
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	protoc \
		--descriptor_set_out=.descriptors.pb \
		--include_imports \
		--proto_path=$(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(XATU_CBT_DIR)/pkg/proto/clickhouse/*.proto \
		$(XATU_CBT_DIR)/pkg/proto/clickhouse/clickhouse/*.proto
	@echo "$(GREEN)✓ Protobuf descriptors generated: .descriptors.pb$(RESET)"

.generate-server: .openapi .generate-descriptors
	@echo "$(CYAN)==> Generating server interface from OpenAPI spec...$(RESET)"
	@mkdir -p internal/handlers
	@oapi-codegen --config oapi-codegen.yaml openapi.yaml > internal/handlers/generated.go
	@echo "$(CYAN)==> Post-processing generated code...$(RESET)"
	@go run $(POSTPROCESS_TOOL) --input internal/handlers/generated.go
	@echo "$(GREEN)✓ Server interface generated: internal/handlers/generated.go$(RESET)"
	@echo "$(CYAN)==> Generating server implementation...$(RESET)"
	@mkdir -p internal/server
	@go run ./cmd/tools/generate-implementation \
		--openapi openapi.yaml \
		--proto-path $(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--output internal/server/implementation.go
	@echo "$(CYAN)==> Copying OpenAPI spec for embedding...$(RESET)"
	@cp openapi.yaml internal/server/openapi.yaml
	@echo "$(GREEN)✓ Server implementation generated: internal/server/implementation.go$(RESET)"

# Clean generated files and build artifacts
clean:
	@echo "$(CYAN)==> Cleaning generated files...$(RESET)"
	@rm -rf $(TMP_DIR)
	@rm -f $(OUTPUT_FILE)
	@rm -f .descriptors.pb
	@rm -rf bin/
	@rm -f internal/handlers/generated.go
	@rm -f internal/server/implementation.go
	@rm -f internal/server/openapi.yaml
	@rm -rf $(XATU_CBT_DIR)
	@echo "$(GREEN)✓ Cleaned$(RESET)"

# Format Go code
fmt:
	@echo "$(CYAN)==> Formatting Go code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✓ Formatted$(RESET)"

# Run linters
lint:
	@echo "$(CYAN)==> Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, skipping...$(RESET)"; \
	fi

# Export test data from production ClickHouse
# Auto-detects tables from openapi.yaml (any table exposed via API)
export-test-data:
	@echo "$(CYAN)==> Exporting test data from $(TESTDATA_EXPORT_HOST) database $(TESTDATA_EXPORT_DATABASE)...$(RESET)"
	@mkdir -p $(TESTDATA_DIR)
	@echo "$(CYAN)==> Auto-detecting tables from openapi.yaml...$(RESET)"
	@for table in $$(grep -oE '/api/v1/[a-z_0-9]+' openapi.yaml | sed 's|/api/v1/||' | sort -u); do \
		echo "$(CYAN)  -> Exporting $$table...$(RESET)"; \
		curl -sS "$(TESTDATA_EXPORT_HOST)" \
			--data-binary "SELECT * FROM $(TESTDATA_EXPORT_DATABASE).$$table FINAL LIMIT 2 FORMAT JSON" \
			-o "$(TESTDATA_DIR)/$$table.json" || echo "$(YELLOW)  ⚠️  Failed to export $$table (may be empty or inaccessible)$(RESET)"; \
	done
	@echo "$(GREEN)✓ Test data exported to $(TESTDATA_DIR)/$(RESET)"

# Run all tests (unit + integration)
test: unit-test integration-test
	@echo "$(GREEN)✓ All tests passed$(RESET)"

# Run unit tests only (excludes integration tests)
unit-test:
	@echo "$(CYAN)==> Running unit tests...$(RESET)"
	@go install gotest.tools/gotestsum@latest
	@gotestsum --raw-command go test -v -race -failfast -coverprofile=coverage.out -covermode=atomic -json $$(go list ./... | grep -v /integrationtest) && \
		echo "$(GREEN)✓ Unit tests passed$(RESET)"

# Run integration tests
integration-test:
	@echo "$(CYAN)==> Running integration tests...$(RESET)"
	@bash -c "set -o pipefail; go test -v -race -timeout=5m ./internal/integrationtest/... | tee integration-test.log" && \
		echo "$(GREEN)✓ Integration tests passed$(RESET)"
