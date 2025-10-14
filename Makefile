.PHONY: help install-tools proto generate build build-binary run clean fmt lint test unit-test integration-test

# Colors for output (use printf for cross-platform compatibility)
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
RESET := \033[0m

# Config file (can be overridden for tests: make proto CONFIG_FILE=config.test.yaml)
CONFIG_FILE ?= config.yaml

# Read configuration from config file (only if it exists)
ifneq (,$(wildcard $(CONFIG_FILE)))
CLICKHOUSE_DSN := $(shell yq eval '.clickhouse.dsn' $(CONFIG_FILE))
CLICKHOUSE_DB := $(shell yq eval '.clickhouse.database' $(CONFIG_FILE))
DISCOVERY_PREFIXES := $(shell yq eval '.clickhouse.discovery.prefixes | join(",")' $(CONFIG_FILE))
DISCOVERY_EXCLUDE := $(shell yq eval '(.clickhouse.discovery.exclude // []) | join(",")' $(CONFIG_FILE))

# Proto generation settings
PROTO_OUTPUT := $(shell yq eval '.proto.output_dir' $(CONFIG_FILE))
PROTO_PACKAGE := $(shell yq eval '.proto.package' $(CONFIG_FILE))
PROTO_GO_PACKAGE := $(shell yq eval '.proto.go_package' $(CONFIG_FILE))
PROTO_INCLUDE_COMMENTS := $(shell yq eval '.proto.include_comments' $(CONFIG_FILE))

# API settings
API_BASE_PATH := $(shell yq eval '.api.base_path' $(CONFIG_FILE))
API_EXPOSE_PREFIXES := $(shell yq eval '.api.expose_prefixes | join(",")' $(CONFIG_FILE))
endif

# Paths
PROTO_PATH := $(PROTO_OUTPUT)
TMP_DIR := ./tmp
OUTPUT_FILE := ./openapi.yaml
PREPROCESS_TOOL := ./cmd/tools/openapi-preprocess
POSTPROCESS_TOOL := ./cmd/tools/openapi-postprocess

# Vendored googleapis path
GOOGLEAPIS_DIR := ./third_party/googleapis

.DEFAULT_GOAL := help

help: ## Show this help message
	@printf "$(CYAN)cbt-api Makefile$(RESET)\n"
	@printf "\n"
	@printf "$(GREEN)Main workflow:$(RESET)\n"
	@printf "  $(CYAN)make install-tools$(RESET)  # One-time setup: install required dependencies\n"
	@printf "  $(CYAN)make proto$(RESET)          # Generate Protocol Buffers from ClickHouse schema\n"
	@printf "  $(CYAN)make generate$(RESET)       # Generate OpenAPI spec and server code from protos\n"
	@printf "  $(CYAN)make build$(RESET)          # Generate all code + build server binary (proto + generate + binary)\n"
	@printf "  $(CYAN)make build-binary$(RESET)   # Build the API server binary only (no code generation)\n"
	@printf "  $(CYAN)make run$(RESET)            # Run the API server\n"
	@printf "\n"
	@printf "$(GREEN)Development:$(RESET)\n"
	@printf "  $(CYAN)make clean$(RESET)          # Remove generated files and build artifacts\n"
	@printf "  $(CYAN)make fmt$(RESET)            # Format Go code\n"
	@printf "  $(CYAN)make lint$(RESET)           # Run linters\n"
	@printf "  $(CYAN)make test$(RESET)           # Run all tests (always cleans + regenerates from test schema)\n"
	@printf "\n"
	@printf "$(GREEN)Testing:$(RESET)\n"
	@printf "  $(CYAN)make unit-test$(RESET)      # Run unit tests only\n"
	@printf "  $(CYAN)make integration-test$(RESET) # Run integration tests only (requires ClickHouse + seeded data)\n"

# Install required development tools (one-time setup)
install-tools:
	@printf "$(CYAN)==> Installing required tools...$(RESET)\n"
	@if ! command -v yq >/dev/null 2>&1; then \
		printf "$(YELLOW)==> Installing yq...$(RESET)\n"; \
		if [ "$$(uname)" = "Linux" ]; then \
			if command -v apk >/dev/null 2>&1; then \
				apk add --no-cache yq; \
			elif command -v apt-get >/dev/null 2>&1; then \
				if [ "$$(id -u)" -eq 0 ]; then \
					apt-get update && apt-get install -y wget && \
					wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 && \
					chmod +x /usr/local/bin/yq; \
				else \
					sudo apt-get update && sudo apt-get install -y wget && \
					sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 && \
					sudo chmod +x /usr/local/bin/yq; \
				fi; \
			else \
				printf "$(RED)Error: No supported package manager found$(RESET)\n"; \
				exit 1; \
			fi; \
		elif [ "$$(uname)" = "Darwin" ]; then \
			brew install yq; \
		else \
			printf "$(RED)Error: Unsupported OS$(RESET)\n"; \
			exit 1; \
		fi; \
	else \
		printf "$(GREEN)✓ yq already installed$(RESET)\n"; \
	fi
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
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install github.com/kollalabs/protoc-gen-openapi@latest
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go get github.com/getkin/kin-openapi/openapi3@latest
	@go get gopkg.in/yaml.v3@latest
	@printf "$(GREEN)✓ Tools installed$(RESET)\n"

# Download googleapis proto files (one-time, vendored locally)
.download-googleapis:
	@if [ ! -d "$(GOOGLEAPIS_DIR)" ]; then \
		printf "$(CYAN)==> Downloading googleapis protos...$(RESET)\n"; \
		mkdir -p third_party; \
		git clone --depth 1 --filter=blob:none --sparse \
			https://github.com/googleapis/googleapis.git $(GOOGLEAPIS_DIR); \
		cd $(GOOGLEAPIS_DIR) && git sparse-checkout set google/api; \
		printf "$(GREEN)✓ Downloaded googleapis to $(GOOGLEAPIS_DIR)$(RESET)\n"; \
	else \
		printf "$(GREEN)✓ googleapis already downloaded$(RESET)\n"; \
	fi

# Generate all code (OpenAPI spec + server implementation)
generate: .build-tools .openapi .generate-descriptors .generate-server
	@printf "$(GREEN)✓ Code generation complete$(RESET)\n"

# Generate Protocol Buffers from ClickHouse (separate target for user control)
proto: .discover-tables .proto
	@printf "$(GREEN)✓ Proto generation complete$(RESET)\n"

# Version information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
DIRTY_SUFFIX := $(shell git diff --quiet || echo "-dirty")

# Build ldflags
LDFLAGS := -s -w \
	-X github.com/ethpandaops/cbt-api/internal/version.Release=$(VERSION)-$(GIT_COMMIT)$(DIRTY_SUFFIX) \
	-X github.com/ethpandaops/cbt-api/internal/version.GitCommit=$(GIT_COMMIT)

# Build everything: generate all code + build binary (useful for CI)
build: proto generate build-binary
	@printf "$(GREEN)✓ Build complete$(RESET)\n"

# Build the API server binary only
build-binary:
	@printf "$(CYAN)==> Building API server...$(RESET)\n"
	@printf "$(CYAN)    Version: $(VERSION)-$(GIT_COMMIT)$(DIRTY_SUFFIX)$(RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server
	@printf "$(GREEN)✓ Server built: bin/server$(RESET)\n"

# Run the API server
run: build-binary
	@printf "$(CYAN)==> Starting API server...$(RESET)\n"
	@./bin/server

# Internal targets (not meant to be called directly)
.discover-tables:
	@printf "$(CYAN)==> Discovering tables from ClickHouse...$(RESET)\n"
	@CH_DSN=$$(yq eval '.clickhouse.dsn' $(CONFIG_FILE)); \
	CH_DB=$$(yq eval '.clickhouse.database' $(CONFIG_FILE)); \
	DISCOVERY_PREFIXES=$$(yq eval '.clickhouse.discovery.prefixes | join(",")' $(CONFIG_FILE)); \
	DISCOVERY_EXCLUDE=$$(yq eval '(.clickhouse.discovery.exclude // []) | join(",")' $(CONFIG_FILE)); \
	PREFIX_CONDITIONS=$$(echo "$$DISCOVERY_PREFIXES" | tr ',' '\n' | sed "s/^/name LIKE '/; s/$$/_%%'/" | paste -sd'|' - | sed 's/|/ OR /g'); \
	EXCLUDE_CONDITIONS=$$(echo "$$DISCOVERY_EXCLUDE" | tr ',' '\n' | sed "s/\*/%%/g; s/^/name NOT LIKE '/; s/$$/'/" | paste -sd'&' - | sed 's/&/ AND /g'); \
	QUERY="SELECT arrayStringConcat(groupArray(name), ',') FROM system.tables WHERE database = '$$CH_DB' AND ($$PREFIX_CONDITIONS)"; \
	if [ -n "$$EXCLUDE_CONDITIONS" ]; then QUERY="$$QUERY AND ($$EXCLUDE_CONDITIONS)"; fi; \
	CH_PROTO=$$(echo "$$CH_DSN" | sed 's|^\([^:]*\)://.*|\1|'); \
	CH_HOST=$$(echo "$$CH_DSN" | sed 's|.*://[^@]*@\([^:/]*\).*|\1|'); \
	CH_PORT=$$(echo "$$CH_DSN" | sed -n 's|.*://[^@]*@[^:]*:\([0-9][0-9]*\)[^0-9].*|\1|p'); \
	CH_USER=$$(echo "$$CH_DSN" | sed 's|.*://\([^:]*\):.*|\1|'); \
	CH_PASS=$$(echo "$$CH_DSN" | sed 's|.*://[^:]*:\([^@]*\)@.*|\1|'); \
	if [ "$$CH_PROTO" = "https" ]; then \
		if [ -z "$$CH_PORT" ]; then CH_PORT=443; fi; \
		CH_URL="https://$$CH_HOST:$$CH_PORT"; \
	elif [ "$$CH_PROTO" = "http" ]; then \
		if [ -z "$$CH_PORT" ]; then CH_PORT=80; fi; \
		CH_URL="http://$$CH_HOST:$$CH_PORT"; \
	else \
		CH_URL="http://$$CH_HOST:8123"; \
	fi; \
	TABLES=$$(curl -fsSL "$$CH_URL/?database=$$CH_DB" \
	  --user "$$CH_USER:$$CH_PASS" \
	  --data-binary "$$QUERY FORMAT TSVRaw" 2>&1); \
	CURL_EXIT=$$?; \
	if [ $$CURL_EXIT -ne 0 ] || [ -z "$$TABLES" ] || echo "$$TABLES" | grep -qi "error\|exception\|code:\|invalid\|password"; then \
		printf "$(RED)Error from ClickHouse:$(RESET)\n$$TABLES\n"; \
		printf "$(YELLOW)⚠️  No tables discovered. Please check your credentials and ensure ClickHouse is accessible.$(RESET)\n"; \
		exit 1; \
	fi; \
	echo $$TABLES > .tables.txt; \
	printf "$(GREEN)✓ Discovered tables: $$TABLES$(RESET)\n"

.proto: .discover-tables .download-googleapis
	@printf "$(CYAN)==> Generating Protocol Buffers from ClickHouse...$(RESET)\n"
	@TABLES=$$(cat .tables.txt); \
	CH_DSN=$$(yq eval '.clickhouse.dsn' $(CONFIG_FILE)); \
	CH_DB=$$(yq eval '.clickhouse.database' $(CONFIG_FILE)); \
	PROTO_OUT=$$(yq eval '.proto.output_dir' $(CONFIG_FILE)); \
	PROTO_PKG=$$(yq eval '.proto.package' $(CONFIG_FILE)); \
	PROTO_GO_PKG=$$(yq eval '.proto.go_package' $(CONFIG_FILE)); \
	PROTO_COMMENTS=$$(yq eval '.proto.include_comments' $(CONFIG_FILE)); \
	API_BASE=$$(yq eval '.api.base_path' $(CONFIG_FILE)); \
	API_PREFIXES=$$(yq eval '.api.expose_prefixes | join(",")' $(CONFIG_FILE)); \
	NATIVE_DSN="$$CH_DSN/$$CH_DB"; \
	if echo "$$NATIVE_DSN" | grep -q "^https://"; then \
		NATIVE_DSN="$$NATIVE_DSN?secure=true"; \
	fi; \
	NETWORK_FLAG=""; \
	if echo "$$NATIVE_DSN" | grep -Eq "localhost|127\.0\.0\.1"; then \
		NATIVE_DSN=$$(echo "$$NATIVE_DSN" | sed 's|localhost|clickhouse|g' | sed 's|127\.0\.0\.1|clickhouse|g'); \
		NETWORK_FLAG="--network examples_default"; \
	fi; \
	docker pull ethpandaops/clickhouse-proto-gen:latest; \
	docker run --rm -v "$$(pwd):/workspace" \
	  --user "$$(id -u):$$(id -g)" \
	  $$NETWORK_FLAG \
	  ethpandaops/clickhouse-proto-gen \
	  --dsn "$$NATIVE_DSN" \
	  --tables "$$TABLES" \
	  --out /workspace/$$PROTO_OUT \
	  --package $$PROTO_PKG \
	  --go-package $$PROTO_GO_PKG \
	  --include-comments=$$PROTO_COMMENTS \
	  --enable-api \
	  --api-table-prefixes $$API_PREFIXES \
	  --api-base-path $$API_BASE \
	  --verbose \
	  --debug
	@printf "$(CYAN)==> Compiling proto files to Go...$(RESET)\n"
	@PROTO_OUT=$$(yq eval '.proto.output_dir' $(CONFIG_FILE)); \
	protoc \
		--go_out=$$PROTO_OUT \
		--go_opt=paths=source_relative \
		--proto_path=$$PROTO_OUT \
		--proto_path=$(GOOGLEAPIS_DIR) \
		$$PROTO_OUT/*.proto \
		$$PROTO_OUT/clickhouse/*.proto
	@PROTO_OUT=$$(yq eval '.proto.output_dir' $(CONFIG_FILE)); \
	printf "$(GREEN)✓ Proto files generated and compiled in $$PROTO_OUT$(RESET)\n"

.build-tools:
	@printf "$(CYAN)==> Building code generation tools...$(RESET)\n"
	@go build -o bin/openapi-preprocess $(PREPROCESS_TOOL)
	@printf "$(GREEN)✓ Built: bin/openapi-preprocess$(RESET)\n"

.openapi: .build-tools .generate-descriptors .download-googleapis
	@printf "$(CYAN)==> Generating OpenAPI 3.0 from annotated protos...$(RESET)\n"
	@mkdir -p $(TMP_DIR)
	@protoc \
		--openapi_out=$(TMP_DIR) \
		--openapi_opt=naming=proto \
		--proto_path=$(PROTO_PATH) \
		--proto_path=$(GOOGLEAPIS_DIR) \
		$(PROTO_PATH)/*.proto
	@printf "$(GREEN)✓ OpenAPI spec generated: $(OUTPUT_FILE)$(RESET)\n"
	@printf "$(CYAN)==> Pre-processing OpenAPI spec...$(RESET)\n"
	@go run $(PREPROCESS_TOOL) \
		--input $(TMP_DIR)/openapi.yaml \
		--output $(OUTPUT_FILE) \
		--proto-path $(PROTO_PATH) \
		--descriptor .descriptors.pb \
		--config $(CONFIG_FILE)
	@printf "$(GREEN)✓ Pre-processed spec generated: $(OUTPUT_FILE)$(RESET)\n"

.generate-descriptors: .download-googleapis
	@printf "$(CYAN)==> Generating protobuf descriptors...$(RESET)\n"
	@protoc \
		--descriptor_set_out=.descriptors.pb \
		--include_imports \
		--proto_path=$(PROTO_OUTPUT) \
		--proto_path=$(GOOGLEAPIS_DIR) \
		$(PROTO_OUTPUT)/*.proto \
		$(PROTO_OUTPUT)/clickhouse/*.proto
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
		--proto-path $(PROTO_OUTPUT) \
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
	@rm -f .tables.txt
	@rm -rf bin/
	@rm -f internal/handlers/generated.go
	@rm -f internal/server/implementation.go
	@rm -f internal/server/openapi.yaml
	@rm -f /tmp/cbt-api-test.log /tmp/cbt-api-test.pid config.test.yaml
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

# Internal: Start test ClickHouse with schema and seed data
.start-test-clickhouse:
	@printf "$(CYAN)==> Starting test ClickHouse...$(RESET)\n"
	@docker compose -f examples/docker-compose.yml down -v 2>/dev/null || true
	@docker compose -f examples/docker-compose.yml up -d
	@printf "$(CYAN)==> Waiting for ClickHouse to be healthy...$(RESET)\n"
	@timeout 60 bash -c 'until [ "$$(docker inspect -f {{.State.Health.Status}} cbt-api-clickhouse 2>/dev/null)" = "healthy" ]; do sleep 1; done'
	@printf "$(CYAN)==> Removing network restriction...$(RESET)\n"
	@docker exec cbt-api-clickhouse rm -f /etc/clickhouse-server/users.d/default-user.xml
	@docker exec cbt-api-clickhouse clickhouse-client --query "SYSTEM RELOAD CONFIG"
	@sleep 2
	@printf "$(CYAN)==> Loading example schema...$(RESET)\n"
	@for table_file in examples/table_*.sql; do \
		cat "$$table_file" | curl -X POST "http://localhost:8123/?database=testdb" --user default: --data-binary @- || exit 1; \
	done
	@printf "$(CYAN)==> Seeding test data...$(RESET)\n"
	@for seed_file in examples/seed_*.sql; do \
		cat "$$seed_file" | curl -X POST "http://localhost:8123/?database=testdb" --user default: --data-binary @- || exit 1; \
	done
	@printf "$(GREEN)✓ Test ClickHouse ready$(RESET)\n"

# Internal: Create test configuration file
.create-test-config:
	@printf "$(CYAN)==> Creating config.test.yaml...$(RESET)\n"
	@printf "%s\n" "clickhouse:" > config.test.yaml
	@printf "%s\n" "  dsn: \"clickhouse://default:@localhost:9000\"" >> config.test.yaml
	@printf "%s\n" "  database: \"testdb\"" >> config.test.yaml
	@printf "%s\n" "  discovery:" >> config.test.yaml
	@printf "%s\n" "    prefixes:" >> config.test.yaml
	@printf "%s\n" "      - fct" >> config.test.yaml
	@printf "%s\n" "" >> config.test.yaml
	@printf "%s\n" "proto:" >> config.test.yaml
	@printf "%s\n" "  output_dir: \"./pkg/proto/clickhouse\"" >> config.test.yaml
	@printf "%s\n" "  package: \"cbt.v1\"" >> config.test.yaml
	@printf "%s\n" "  go_package: \"github.com/ethpandaops/cbt-api/pkg/proto/clickhouse\"" >> config.test.yaml
	@printf "%s\n" "  include_comments: true" >> config.test.yaml
	@printf "%s\n" "" >> config.test.yaml
	@printf "%s\n" "api:" >> config.test.yaml
	@printf "%s\n" "  base_path: \"/api/v1\"" >> config.test.yaml
	@printf "%s\n" "  expose_prefixes:" >> config.test.yaml
	@printf "%s\n" "    - fct" >> config.test.yaml
	@printf "%s\n" "" >> config.test.yaml
	@printf "%s\n" "server:" >> config.test.yaml
	@printf "%s\n" "  port: 18080" >> config.test.yaml
	@printf "%s\n" "  host: \"0.0.0.0\"" >> config.test.yaml
	@printf "$(GREEN)✓ Test config created$(RESET)\n"

# Internal: Setup for linting (generates code from test schema)
# Used by CI linting to generate files without requiring production ClickHouse
.lint-setup:
	@printf "$(CYAN)==> Generating code from test schema for linting...$(RESET)\n"
	@$(MAKE) .start-test-clickhouse
	@$(MAKE) .create-test-config
	@$(MAKE) proto generate CONFIG_FILE=config.test.yaml
	@printf "$(GREEN)✓ Lint setup complete$(RESET)\n"

# Internal: Clean up test environment (ClickHouse + test config)
.cleanup-test-env:
	@printf "$(CYAN)==> Cleaning up test environment...$(RESET)\n"
	@docker compose -f examples/docker-compose.yml down -v 2>/dev/null || true
	@rm -f config.test.yaml
	@printf "$(GREEN)✓ Test environment cleaned$(RESET)\n"

# Run all tests (always cleans and regenerates from test schema)
test: clean
	@printf "$(CYAN)==> Setting up test environment...$(RESET)\n"
	@$(MAKE) .start-test-clickhouse
	@$(MAKE) .create-test-config
	@printf "$(CYAN)==> Generating code from test schema...$(RESET)\n"
	@$(MAKE) proto generate CONFIG_FILE=config.test.yaml
	@printf "$(CYAN)==> Running tests...$(RESET)\n"
	@$(MAKE) unit-test integration-test
	@$(MAKE) .cleanup-test-env
	@printf "$(GREEN)✓ All tests passed$(RESET)\n"

# Run unit tests only
unit-test:
	@printf "$(CYAN)==> Running unit tests...$(RESET)\n"
	@go install gotest.tools/gotestsum@latest
	@gotestsum --raw-command go test -v -race -failfast -coverprofile=coverage.out -covermode=atomic -json $$(go list ./...) && \
		printf "$(GREEN)✓ Unit tests passed$(RESET)\n"

# Run integration tests (requires ClickHouse to be running with seeded data)
integration-test: build-binary
	@printf "$(CYAN)==> Running integration tests...$(RESET)\n"
	@if [ ! -f "config.test.yaml" ]; then \
		printf "$(YELLOW)⚠️  config.test.yaml not found, creating it...$(RESET)\n"; \
		$(MAKE) .create-test-config; \
	fi
	@printf "$(CYAN)==> Starting API server in background...$(RESET)\n"
	@./bin/server --config config.test.yaml > /tmp/cbt-api-test.log 2>&1 & \
	SERVER_PID=$$!; \
	echo $$SERVER_PID > /tmp/cbt-api-test.pid; \
	printf "$(CYAN)==> Waiting for server to be ready (PID: $$SERVER_PID)...$(RESET)\n"; \
	MAX_RETRIES=30; \
	RETRY=0; \
	while [ $$RETRY -lt $$MAX_RETRIES ]; do \
		if curl -sf http://localhost:18080/health >/dev/null 2>&1; then \
			printf "$(GREEN)✓ Server is ready$(RESET)\n"; \
			break; \
		fi; \
		RETRY=$$((RETRY + 1)); \
		if [ $$RETRY -eq $$MAX_RETRIES ]; then \
			printf "$(RED)✗ Server failed to start within timeout$(RESET)\n"; \
			cat /tmp/cbt-api-test.log; \
			kill $$SERVER_PID 2>/dev/null || true; \
			rm -f /tmp/cbt-api-test.pid; \
			exit 1; \
		fi; \
		sleep 1; \
	done; \
	printf "$(CYAN)==> Testing API endpoints...$(RESET)\n"; \
	FAILED=0; \
	for endpoint in fct_data_types_integers fct_data_types_temporal fct_data_types_complex; do \
		printf "$(CYAN)  Testing /api/v1/$$endpoint...$(RESET) "; \
		HTTP_CODE=$$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:18080/api/v1/$$endpoint?id_eq=1"); \
		if [ "$$HTTP_CODE" = "404" ]; then \
			printf "$(RED)✗ $$HTTP_CODE (endpoint not found)$(RESET)\n"; \
			FAILED=1; \
		elif [ "$$HTTP_CODE" = "200" ]; then \
			printf "$(GREEN)✓ $$HTTP_CODE$(RESET)\n"; \
		else \
			printf "$(YELLOW)⚠ $$HTTP_CODE (endpoint exists, may have validation/query errors)$(RESET)\n"; \
		fi; \
	done; \
	printf "$(CYAN)==> Stopping API server...$(RESET)\n"; \
	kill $$SERVER_PID 2>/dev/null || true; \
	rm -f /tmp/cbt-api-test.pid; \
	if [ $$FAILED -eq 1 ]; then \
		printf "$(RED)✗ Integration tests failed$(RESET)\n"; \
		cat /tmp/cbt-api-test.log; \
		exit 1; \
	fi; \
	printf "$(GREEN)✓ Integration tests passed$(RESET)\n"
