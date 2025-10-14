# cbt-api

Generates OpenAPI spec and server implementation for ClickHouse-based REST APIs using CBT (ClickHouse Build Tool).

## Overview

cbt-api is a **generic REST API generator** for any ClickHouse database managed with [CBT](https://github.com/ethpandaops/cbt). It automatically:
- Discovers tables from your ClickHouse database
- Generates Protocol Buffer definitions from table schemas
- Creates a complete OpenAPI 3.0 specification
- Generates a fully functional Go REST API server

**What it generates:**
- `pkg/proto/clickhouse/*.proto` - Protocol Buffer definitions from ClickHouse tables
- `openapi.yaml` - OpenAPI 3.0 specification with flattened filter parameters
- `internal/handlers/generated.go` - Server interface (via oapi-codegen)
- `internal/server/implementation.go` - Complete server implementation with automatic query building

## Quick Start

```bash
# 1. Configure your ClickHouse connection
cp config.example.yaml config.yaml
# Edit config.yaml with your ClickHouse DSN and database

# 2. One-time setup: install dependencies
make install-tools

# 3. Generate proto files from your ClickHouse tables
make proto

# 4. Generate OpenAPI spec + server code
make generate

# 5. Build and run the server
make run
```

Visit `http://localhost:8080/docs` to explore the API via Swagger UI.

## Make Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `make help` | Show available commands |
| `make install-tools` | One-time setup: install required dependencies |
| `make proto` | Generate proto files from ClickHouse tables |
| `make generate` | Generate OpenAPI spec and server code |
| `make build` | Build the API server binary |
| `make run` | Build and run the API server |
| `make clean` | Remove generated files and build artifacts |
| `make fmt` | Format Go code |
| `make lint` | Run linters |

### Testing Commands

| Command | Description |
|---------|-------------|
| `make test` | Run unit tests |
| `make unit-test` | Run unit tests only |

## Configuration

Copy `config.example.yaml` to `config.yaml` and configure:

### ClickHouse Connection

```yaml
clickhouse:
  dsn: "https://user:password@host:443"
  database: "mainnet"
  use_final: false  # Add FINAL modifier to queries

  # Table discovery for proto generation
  discovery:
    prefixes:
      - fct        # Discover fact tables
      - dim        # Discover dimension tables
    exclude:
      - "*_test"   # Exclude test tables
      - "*_tmp"    # Exclude temporary tables
```

### Proto Generation Settings

```yaml
proto:
  output_dir: "./pkg/proto/clickhouse"
  package: "cbt.v1"
  go_package: "github.com/ethpandaops/cbt-api/pkg/proto/clickhouse"
  include_comments: true
```

### API Exposure Settings

```yaml
api:
  enable: true
  base_path: "/api/v1"
  # Only tables with these prefixes will be exposed via REST API
  expose_prefixes:
    - fct  # Expose fact tables
```

### Telemetry (Optional)

```yaml
telemetry:
  enabled: true
  endpoint: "tempo.example.com:443"
  service_name: "cbt-api"
  sample_rate: 0.1  # 10% sampling
  always_sample_errors: true
```

## API Overview

### Endpoints

All tables matching `api.expose_prefixes` are exposed as REST endpoints:

```
GET /api/v1/fct_block
GET /api/v1/fct_attestation_correctness_by_validator_head
GET /api/v1/fct_mev_bid_count_by_builder
...
```

Each table gets two operations:
- **List** - Query with filters, pagination, sorting (`GET /api/v1/{table}`)
- **Get** - Retrieve by primary key (if available)

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

### Pagination

```
?page_size=100          # Items per page (default: 100, max: 10000)
?page_token=offset_500  # Continue from previous page
```

### Sorting

```
?order_by=slot          # Sort by field
?order_desc=true        # Descending order
```

## How It Works

### Generation Pipeline

1. **Table Discovery** (`make proto`)
   - Queries ClickHouse `system.tables` to find tables matching configured prefixes
   - Generates proto files using [clickhouse-proto-gen](https://github.com/ethpandaops/clickhouse-proto-gen)
   - Creates Protocol Buffer definitions from table schemas
   - Generates query builder functions for each table

2. **OpenAPI Generation** (`make generate`)
   - Generates `.descriptors.pb` from proto files
   - Creates `openapi.yaml` from proto annotations (HTTP mappings)
   - Flattens nested filter parameters for REST-friendly URLs
   - Generates server interface via oapi-codegen

3. **Implementation Generation**
   - Analyzes proto descriptors + OpenAPI spec
   - Generates complete server implementation
   - Maps HTTP parameters to proto request types
   - Integrates with generated query builders

### Request Flow

```
HTTP Request
  ↓
oapi-codegen Router (validates params)
  ↓
Generated Handler (maps HTTP → Proto)
  ↓
Query Builder (generates SQL from proto filters)
  ↓
ClickHouse Execution
  ↓
Result Scanning (Proto types)
  ↓
Type Conversion (Proto → OpenAPI)
  ↓
JSON Response
```

### Example: Generated Handler

```go
// internal/server/implementation.go (auto-generated)
func (s *Server) FctBlockServiceList(w http.ResponseWriter, r *http.Request, params handlers.FctBlockServiceListParams) {
    // 1. Map HTTP params → Proto request
    req := &clickhouse.ListFctBlockRequest{PageSize: 100}
    if params.SlotEq != nil {
        req.Slot = &clickhouse.UInt32Filter{
            Filter: &clickhouse.UInt32Filter_Eq{Eq: uint32(*params.SlotEq)},
        }
    }

    // 2. Build SQL query from proto request
    sqlQuery, _ := clickhouse.BuildListFctBlockQuery(req, queryOpts...)

    // 3. Execute on ClickHouse
    rows, _ := s.db.Query(ctx, sqlQuery.Query, sqlQuery.Args...)

    // 4. Scan results and convert Proto → OpenAPI
    // 5. Return JSON response
}
```

## Development

### Updating Table Schemas

When your ClickHouse tables change:

```bash
# Re-generate proto files from updated schemas
make proto

# Re-generate server code
make generate

# Test the changes
make test
```

### Adding New Tables

1. Update `config.yaml` to include new table prefixes in `discovery.prefixes`
2. Run `make proto` to generate protos for new tables
3. Run `make generate` to expose new tables via REST API

### Excluding Tables

Add patterns to `config.yaml`:

```yaml
clickhouse:
  discovery:
    exclude:
      - "*_test"
      - "*_tmp"
      - "*_staging"
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make unit-test
```

## Using with Different CBT Projects

This project is **generic** and can be used with any CBT-managed ClickHouse database:

1. **Fork or clone** this repository
2. **Update `config.yaml`** with your ClickHouse connection and table prefixes
3. **Run `make proto`** to generate proto files from your tables
4. **Run `make generate`** to create your REST API

The generated API will automatically expose all tables matching your configured prefixes with full query capabilities.

## Server Endpoints

- **API endpoints** at `/api/v1/*`
- **Health check** at `/health`
- **Metrics** at `/metrics` (Prometheus format)
- **OpenAPI spec** at `/openapi.yaml`
- **Swagger UI** at `/docs/`

## Dependencies

Install with `make install-tools`.
