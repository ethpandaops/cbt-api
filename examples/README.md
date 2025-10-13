# Example Schema for xatu-cbt-api

## Overview

This directory contains a minimal ClickHouse schema for:
- **CI Testing**: GitHub Actions uses this schema to test proto generation without production credentials
- **Local Development**: Developers can test the generation pipeline without setting up a full database
- **Getting Started**: New users can see a working example before connecting their own ClickHouse

The example schema includes 3 generic tables testing comprehensive ClickHouse data types:
- `fct_data_types_integers` - All integer types including UInt128/256, Int64/128
- `fct_data_types_temporal` - DateTime, DateTime64, Date types with timezone handling
- `fct_data_types_complex` - Floats, Decimals, Strings, Arrays, Maps, Enums, UUID, IPv4/6

## Local Development Setup

### Option 1: Docker Compose (Recommended)

Start a local ClickHouse container with the example schema:

```bash
# 1. Start ClickHouse container
docker-compose -f examples/docker-compose.yml up -d

# 2. Wait for it to be ready and load schema
sleep 5
cat examples/test-schema.sql | docker exec -i xatu-cbt-api-clickhouse clickhouse-client

# 3. Create config.yaml pointing to local container
cat > config.yaml << EOF
clickhouse:
  dsn: "clickhouse://default:@localhost:9000/testdb"
  database: "testdb"
  discovery:
    prefixes:
      - fct

proto:
  output_dir: "./pkg/proto/clickhouse"
  package: "cbt.v1"
  go_package: "github.com/ethpandaops/xatu-cbt-api/pkg/proto/clickhouse"
  include_comments: true

api:
  base_path: "/api/v1"
  expose_prefixes:
    - fct
EOF

# 4. Generate protos and build
make install-tools
make proto generate

# 5. Run the server
make run

# Visit http://localhost:8080/docs to see the generated API

# When done, cleanup:
docker-compose -f examples/docker-compose.yml down -v
```

### Option 2: Your Own ClickHouse

To use xatu-cbt-api with your own ClickHouse database:

```bash
# Copy the example config
cp config.example.yaml config.yaml

# Edit config.yaml with your ClickHouse connection details
# Update:
#   - clickhouse.dsn (your connection string)
#   - clickhouse.database (your database name)
#   - clickhouse.discovery.prefixes (your table prefixes)

# Generate protos from YOUR schema
make install-tools
make proto
make generate

# Build and run
make run
```

## CI Usage

GitHub Actions workflows use this example schema to test the proto generation pipeline without requiring production ClickHouse credentials. The CI process:

1. Starts a ClickHouse container (service)
2. Loads `test-schema.sql` into the container
3. Runs `make proto && make generate` against the example database
4. Runs tests/linting against the generated code

This ensures the generation pipeline works correctly while allowing forks and external contributors to run CI without secrets.

## Schema Details

### fct_data_types_integers
Comprehensive integer type testing:
- Primary key: `(id, timestamp)`
- **Unsigned integers**: UInt8, UInt16, UInt32, UInt64, UInt128, UInt256
- **Signed integers**: Int32, Int64, Int128
- **Nullable variants**: Tests google.protobuf wrapper generation
- **Critical test**: UInt128/UInt256 conversion to string (known problematic type)

### fct_data_types_temporal
DateTime and Date type testing:
- Primary key: `(id)`
- **Date types**: Date, Date32 (extended range)
- **DateTime types**: DateTime, DateTime64 with millisecond/microsecond/nanosecond precision
- **Timezone handling**: DateTime('UTC'), DateTime64(3, 'UTC')
- **Critical test**: DateTime conversion to UInt32 (known problematic conversion)
- **Nullable variants**: Tests wrapper types for temporal data

### fct_data_types_complex
Complex and special ClickHouse types:
- Primary key: `(id, timestamp)`
- **Floating point**: Float32, Float64
- **Decimals**: Decimal32(4), Decimal64(8), Decimal128(18) - tests precision handling
- **Strings**: String, FixedString(32), LowCardinality(String)
- **Enums**: Enum8, Enum16
- **Collections**: Array(UInt32), Array(String), Map(String, String)
- **Special types**: UUID, IPv4, IPv6, Bool
- **Nullable variants**: Tests wrapper generation for complex types

## Cleanup

```bash
# Stop and remove container
docker-compose -f examples/docker-compose.yml down -v

# Remove generated files
make clean
```
