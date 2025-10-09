# xatu-cbt-api

Generates OpenAPI spec and server implementation for Xatu CBT REST API.

## Overview

Generates from [xatu-cbt](https://github.com/ethpandaops/xatu-cbt) proto definitions:
- OpenAPI 3.0 specification with flattened filter parameters
- Go server interface (via oapi-codegen)
- Complete server implementation with automatic proto → HTTP mapping

## Quick Start

```bash
make install-tools    # Install dependencies
make generate-server  # Generate OpenAPI spec + server code
make serve-docs       # View in Swagger UI (localhost:3001)
```

Generates:
- `openapi.yaml` - OpenAPI spec
- `internal/handlers/generated.go` - Server interface (oapi-codegen)
- `internal/server/implementation.go` - Server implementation

## Make Commands

### Code Generation

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make all` | Install tools, build, and generate OpenAPI spec and server code |
| `make clone-xatu-cbt` | Clone/update xatu-cbt repository for proto files |
| `make openapi` | Generate OpenAPI specification from proto files |
| `make generate-descriptors` | Generate protobuf descriptor file for robust parsing |
| `make generate-server` | Generate Go server interface and implementation from OpenAPI specification |
| `make build` | Build the openapi-filter-flatten tool |
| `make install-tools` | Install required dependencies (protoc-gen-openapi, oapi-codegen, etc.) |

### Server Runtime

| Command | Description |
|---------|-------------|
| `make build-server` | Build the API server binary (bin/server) |
| `make run-server` | Build and run the API server |

### Development

| Command | Description |
|---------|-------------|
| `make validate` | Validate the generated OpenAPI spec |
| `make serve-docs` | Serve OpenAPI spec with Swagger UI (http://localhost:3001) |
| `make clean` | Remove all generated files and build artifacts |
| `make fmt` | Format Go code |
| `make lint` | Run Go linters |
| `make test` | Run tests |

## API Overview

### Endpoints

All fact tables (`fct_*`) from xatu-cbt are exposed as REST endpoints:

```
GET /api/v1/fct_attestation_correctness_by_validator_head
GET /api/v1/fct_block
GET /api/v1/fct_mev_bid_count_by_builder
...
```

### Filter Parameters

Filters use underscore notation with operator suffixes:

```
?slot_start_date_time_gte=1609459200
?slot_start_date_time_lte=1609545600
?attesting_validator_index_eq=12345
?meta_client_name_in_values=lighthouse,prysm,teku
```

**Supported operators:**
- Scalar: `eq`, `ne`, `lt`, `lte`, `gt`, `gte`, `in_values`, `not_in_values`
- String: `contains`, `starts_with`, `ends_with`, `like`, `not_like`
- Nullable: `is_null`, `is_not_null`
- Map: `has_key`, `not_has_key`, `has_any_key`, `has_all_keys`

**Note:** List filters (`_in_values`, `_not_in_values`) use comma-separated strings.

## Running the Server

### Configuration

Copy `config.example.yaml` to `config.yaml` file.

### Starting the Server

```bash
# Build and run
make all && make run-server

# Or build separately
make build-server
./bin/server
```

The server provides:
- **API endpoints** at `/api/v1/*`
- **Health check** at `/health`
- **Metrics** at `/metrics`
- **OpenAPI spec** at `/openapi.yaml`
- **Swagger UI** at `/docs/`

## Server Implementation

The server implementation is **automatically generated** from proto descriptors:

```go
// Example: internal/server/implementation.go (generated)
func (s *Server) FctBlockServiceList(w http.ResponseWriter, r *http.Request, params handlers.FctBlockServiceListParams) {
    // 1. Map HTTP params → Proto request
    req := &clickhouse.ListFctBlockRequest{PageSize: 100}
    if params.SlotEq != nil {
        req.Slot = &clickhouse.UInt32Filter{
            Filter: &clickhouse.UInt32Filter_Eq{Eq: uint32(*params.SlotEq)},
        }
    }

    // 2. Use xatu-cbt query builder
    sqlQuery, _ := clickhouse.BuildListFctBlockQuery(req)

    // 3. Execute and return results
    rows, _ := s.db.Query(ctx, sqlQuery.Query, sqlQuery.Args...)
    // ... scan and respond
}
```

**What's generated:**
- All xatu-cbt endpoints (List + Get for each table)
- HTTP params → Proto filter mapping
- Query builder integration
- ClickHouse execution + result scanning
- Proto → OpenAPI type conversion

## How It Works

**Generation pipeline:**
1. Proto descriptors → `.descriptors.pb` (via protoc)
2. Proto files → `openapi.yaml` (via protoc-gen-openapi + flattening)
3. OpenAPI → `internal/handlers/generated.go` (via oapi-codegen)
4. Descriptors + OpenAPI → `internal/server/implementation.go` (via generate-implementation)

**Request flow:**
1. HTTP request → oapi-codegen router parses params
2. Generated implementation maps params → proto filters
3. Calls xatu-cbt query builder → SQL query
4. Executes on ClickHouse → scan results
5. Converts proto → OpenAPI types → JSON response

## Development

Update xatu-cbt version in Makefile, then regenerate:
```bash
make clean && make all
```

