.PHONY: help openapi clean build install-tools test fmt lint all clone-xatu-cbt build-server run-server

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
RESET := \033[0m

# xatu-cbt repository configuration
XATU_CBT_REPO := https://github.com/ethpandaops/xatu-cbt.git
XATU_CBT_VERSION ?= 2ea700f24c4480c96e8f58e4c53960dfccdbecda
XATU_CBT_DIR := ./.xatu-cbt

# Paths
PROTO_PATH := $(XATU_CBT_DIR)/pkg/proto/clickhouse
TMP_DIR := ./tmp
OUTPUT_FILE := ./openapi.yaml
PREPROCESS_TOOL := ./cmd/tools/openapi-preprocess
POSTPROCESS_TOOL := ./cmd/tools/openapi-postprocess

# Get googleapis path
GOOGLEAPIS_PATH := $(shell go list -m -f '{{.Dir}}' github.com/googleapis/googleapis 2>/dev/null || echo "")

help: ## Show this help message
	@echo "$(CYAN)xatu-cbt-api Makefile$(RESET)"
	@echo ""
	@echo "$(GREEN)Available targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

all: install-tools build openapi generate-server ## Install tools, build, generate OpenAPI and server code

clone-xatu-cbt: ## Clone xatu-cbt repository for proto files
	@if [ -d "$(XATU_CBT_DIR)" ]; then \
		echo "$(YELLOW)==> Updating xatu-cbt repository...$(RESET)"; \
		cd $(XATU_CBT_DIR) && git fetch origin && git checkout $(XATU_CBT_VERSION) 2>/dev/null || true; \
	else \
		echo "$(CYAN)==> Cloning xatu-cbt repository (commit: $(XATU_CBT_VERSION))...$(RESET)"; \
		git clone $(XATU_CBT_REPO) $(XATU_CBT_DIR) && cd $(XATU_CBT_DIR) && git checkout $(XATU_CBT_VERSION); \
	fi
	@echo "$(GREEN)✓ xatu-cbt repository ready at $(XATU_CBT_VERSION)$(RESET)"

openapi: build clone-xatu-cbt ## Generate OpenAPI specification from proto files
	@echo "$(CYAN)==> Generating OpenAPI 3.0 from annotated protos...$(RESET)"
	@mkdir -p $(TMP_DIR)
	@if [ -z "$(GOOGLEAPIS_PATH)" ]; then \
		echo "$(RED)Error: googleapis not found. Installing...$(RESET)"; \
		go get github.com/googleapis/googleapis@latest; \
	fi
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	echo "$(YELLOW)Using googleapis: $$GOOGLEAPIS_PATH$(RESET)"; \
	protoc \
		--openapi_out=$(TMP_DIR) \
		--openapi_opt=naming=proto \
		--proto_path=$(PROTO_PATH) \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(PROTO_PATH)/*.proto
	@echo "$(CYAN)==> Pre-processing OpenAPI spec...$(RESET)"
	@go run $(PREPROCESS_TOOL) \
		--input $(TMP_DIR)/openapi.yaml \
		--output $(OUTPUT_FILE) \
		--proto-path $(PROTO_PATH)
	@echo "$(GREEN)✓ OpenAPI spec generated: $(OUTPUT_FILE)$(RESET)"

generate-descriptors: clone-xatu-cbt ## Generate protobuf descriptor file for robust parsing
	@echo "$(CYAN)==> Generating protobuf descriptors...$(RESET)"
	@GOOGLEAPIS_PATH=$$(go list -m -f '{{.Dir}}' github.com/googleapis/googleapis); \
	protoc \
		--descriptor_set_out=.descriptors.pb \
		--include_imports \
		--proto_path=$(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--proto_path=$$GOOGLEAPIS_PATH \
		$(XATU_CBT_DIR)/pkg/proto/clickhouse/*.proto
	@echo "$(GREEN)✓ Protobuf descriptors generated: .descriptors.pb$(RESET)"

generate-server: openapi generate-descriptors ## Generate Go server code from OpenAPI specification
	@echo "$(CYAN)==> Generating server interface from OpenAPI spec...$(RESET)"
	@mkdir -p internal/handlers
	@oapi-codegen --config oapi-codegen.yaml openapi.yaml > internal/handlers/generated.go
	@echo "$(CYAN)==> Post-processing generated code (adding ClickHouse tags)...$(RESET)"
	@go run $(POSTPROCESS_TOOL) --input internal/handlers/generated.go
	@echo "$(GREEN)✓ Server interface generated: internal/handlers/generated.go$(RESET)"
	@echo "$(CYAN)==> Generating server implementation...$(RESET)"
	@mkdir -p internal/server
	@go run ./cmd/tools/generate-implementation \
		--openapi openapi.yaml \
		--proto-path $(XATU_CBT_DIR)/pkg/proto/clickhouse \
		--output internal/server/implementation.go
	@echo "$(GREEN)✓ Server implementation generated: internal/server/implementation.go$(RESET)"

build: ## Build the openapi-preprocess tool
	@echo "$(CYAN)==> Building openapi-preprocess tool...$(RESET)"
	@go build -o bin/openapi-preprocess $(PREPROCESS_TOOL)
	@echo "$(GREEN)✓ Built: bin/openapi-preprocess$(RESET)"

install-tools: ## Install required tools (protoc-gen-openapi, oapi-codegen)
	@echo "$(CYAN)==> Installing required tools...$(RESET)"
	@go install github.com/kollalabs/protoc-gen-openapi@latest
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go get github.com/getkin/kin-openapi/openapi3@latest
	@go get github.com/googleapis/googleapis@latest
	@go get gopkg.in/yaml.v3@latest
	@echo "$(GREEN)✓ Tools installed$(RESET)"

deps: install-tools ## Alias for install-tools

clean: ## Remove generated files and build artifacts
	@echo "$(CYAN)==> Cleaning generated files...$(RESET)"
	@rm -rf $(TMP_DIR)
	@rm -f $(OUTPUT_FILE)
	@rm -rf bin/
	@rm -rf internal/handlers/generated.go
	@rm -rf internal/server/implementation.go
	@rm -rf $(XATU_CBT_DIR)
	@echo "$(GREEN)✓ Cleaned$(RESET)"

fmt: ## Format Go code
	@echo "$(CYAN)==> Formatting Go code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✓ Formatted$(RESET)"

lint: ## Run linters
	@echo "$(CYAN)==> Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, skipping...$(RESET)"; \
	fi

test: ## Run tests
	@echo "$(CYAN)==> Running tests...$(RESET)"
	@go test -v ./...

validate: openapi ## Validate the generated OpenAPI spec
	@echo "$(CYAN)==> Validating OpenAPI spec...$(RESET)"
	@if command -v python3 >/dev/null 2>&1; then \
		python3 -c "import yaml; yaml.safe_load(open('$(OUTPUT_FILE)'))" && \
		echo "$(GREEN)✓ OpenAPI spec is valid YAML$(RESET)"; \
	else \
		echo "$(YELLOW)Python3 not found, skipping validation$(RESET)"; \
	fi

serve-docs: openapi ## Serve OpenAPI spec with Swagger UI
	@echo "$(CYAN)==> Starting Swagger UI...$(RESET)"
	@docker run --rm -d -p 3001:8080 \
		-v $(PWD)/$(OUTPUT_FILE):/openapi.yaml \
		-e SWAGGER_JSON=/openapi.yaml \
		--name xatu-cbt-api-docs \
		swaggerapi/swagger-ui >/dev/null
	@echo "$(YELLOW)Waiting for Swagger UI to be ready...$(RESET)"
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if curl -s http://localhost:3001 >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Swagger UI running at http://localhost:3001$(RESET)"; \
			open http://localhost:3001 || xdg-open http://localhost:3001 || echo "$(YELLOW)Please open http://localhost:3001 in your browser$(RESET)"; \
			echo "$(YELLOW)To stop: docker stop xatu-cbt-api-docs$(RESET)"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "$(RED)Failed to start Swagger UI$(RESET)"; exit 1

build-server: ## Build the API server binary
	@echo "$(CYAN)==> Building API server...$(RESET)"
	@go build -o bin/server ./cmd/server
	@echo "$(GREEN)✓ Server built: bin/server$(RESET)"

run-server: build-server ## Build and run the API server
	@echo "$(CYAN)==> Starting API server...$(RESET)"
	@./bin/server

.DEFAULT_GOAL := help
