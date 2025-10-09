# xatu-cbt-api

Generates OpenAPI spec and server implementation for Xatu CBT REST API.

## Overview

Generates from [xatu-cbt](https://github.com/ethpandaops/xatu-cbt) proto definitions:
- OpenAPI 3.0 specification with flattened filter parameters
- Go server interface (via oapi-codegen)
- Complete server implementation with automatic proto → HTTP mapping

## Quick Start

```bash
make install-tools  # One-time setup: install dependencies
make generate       # Generate OpenAPI spec + server code
make run            # Build and run the server
```

Visit `http://localhost:8080/docs` to explore the API via Swagger UI.

Generates:
- `openapi.yaml` - OpenAPI spec
- `internal/handlers/generated.go` - Server interface (oapi-codegen)
- `internal/server/implementation.go` - Server implementation

## Make Commands

| Command | Description |
|---------|-------------|
| `make help` | Show available commands |
| `make install-tools` | One-time setup: install required dependencies |
| `make generate` | Generate OpenAPI spec and server code |
| `make build` | Build the API server binary |
| `make run` | Build and run the API server |
| `make clean` | Remove generated files and build artifacts |
| `make fmt` | Format Go code |
| `make lint` | Run linters |
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

Copy `config.example.yaml` to `config.yaml` and configure your ClickHouse connection:

```yaml
clickhouse:
  dsn: "clickhouse://user:password@localhost:9000"
  database: "mainnet"
```

### Starting the Server

```bash
# One-time setup
make install-tools

# Generate code and run
make generate
make run
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

