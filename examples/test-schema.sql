-- Example ClickHouse schema for CI testing
-- Generic tables testing comprehensive ClickHouse data types
-- Covers edge cases: UInt128, DateTime conversions, Decimal precision, etc.
--
-- Note: Database 'testdb' is created by Docker container via CLICKHOUSE_DB env var
-- This file only contains table definitions to avoid multi-statement issues

-- Table 1: Integer types (including problematic UInt128)
CREATE TABLE IF NOT EXISTS fct_data_types_integers (
    id UInt64 COMMENT 'Primary identifier',
    timestamp UInt32 COMMENT 'Unix timestamp',

    -- Unsigned integers (all sizes)
    uint8_value UInt8 COMMENT 'Unsigned 8-bit integer',
    uint16_value UInt16 COMMENT 'Unsigned 16-bit integer',
    uint32_value UInt32 COMMENT 'Unsigned 32-bit integer',
    uint64_value UInt64 COMMENT 'Unsigned 64-bit integer',
    uint128_value UInt128 COMMENT 'Unsigned 128-bit integer (tests conversion to string)',
    uint256_value UInt256 COMMENT 'Unsigned 256-bit integer (tests large number handling)',

    -- Signed integers
    int32_value Int32 COMMENT 'Signed 32-bit integer',
    int64_value Int64 COMMENT 'Signed 64-bit integer',
    int128_value Int128 COMMENT 'Signed 128-bit integer',

    -- Nullable integer variants
    nullable_uint32 Nullable(UInt32) COMMENT 'Nullable unsigned 32-bit',
    nullable_uint64 Nullable(UInt64) COMMENT 'Nullable unsigned 64-bit',
    nullable_uint128 Nullable(UInt128) COMMENT 'Nullable UInt128 (tests wrapper conversion)',
    nullable_int64 Nullable(Int64) COMMENT 'Nullable signed 64-bit'
)
ENGINE = MergeTree
PRIMARY KEY (id, timestamp)
ORDER BY (id, timestamp)
COMMENT 'Tests all integer types including problematic UInt128/UInt256 conversions';

-- Table 2: Temporal types (DateTime conversions)
CREATE TABLE IF NOT EXISTS fct_data_types_temporal (
    id UInt64 COMMENT 'Primary identifier',

    -- Date and DateTime types
    date_value Date COMMENT 'Date type (YYYY-MM-DD)',
    date32_value Date32 COMMENT 'Extended date range (1900-2299)',
    datetime_value DateTime COMMENT 'DateTime (tests conversion to UInt32)',
    datetime64_millis DateTime64(3) COMMENT 'DateTime with millisecond precision',
    datetime64_micros DateTime64(6) COMMENT 'DateTime with microsecond precision',
    datetime64_nanos DateTime64(9) COMMENT 'DateTime with nanosecond precision',

    -- DateTime with timezone
    datetime_utc DateTime('UTC') COMMENT 'DateTime with explicit UTC timezone',
    datetime64_utc DateTime64(3, 'UTC') COMMENT 'DateTime64 with UTC timezone',

    -- Nullable temporal types
    nullable_date Nullable(Date) COMMENT 'Nullable date',
    nullable_datetime Nullable(DateTime) COMMENT 'Nullable DateTime (tests wrapper)',
    nullable_datetime64 Nullable(DateTime64(3)) COMMENT 'Nullable DateTime64'
)
ENGINE = MergeTree
PRIMARY KEY (id)
ORDER BY (id)
COMMENT 'Tests DateTime types and problematic DateTime to UInt32 conversions';

-- Table 3: Complex types (Floats, Decimals, Strings, Collections, Special types)
CREATE TABLE IF NOT EXISTS fct_data_types_complex (
    id UInt64 COMMENT 'Primary identifier',
    timestamp UInt32 COMMENT 'Unix timestamp',

    -- Floating point types
    float32_value Float32 COMMENT 'Single precision float',
    float64_value Float64 COMMENT 'Double precision float',
    nullable_float64 Nullable(Float64) COMMENT 'Nullable float',

    -- Decimal types (tests precision handling)
    decimal32_value Decimal32(4) COMMENT 'Decimal with 4 decimal places',
    decimal64_value Decimal64(8) COMMENT 'Decimal with 8 decimal places',
    decimal128_value Decimal128(18) COMMENT 'High-precision decimal',
    nullable_decimal Nullable(Decimal64(8)) COMMENT 'Nullable decimal',

    -- String types
    string_value String COMMENT 'Variable length string',
    fixed_string_value FixedString(32) COMMENT 'Fixed-length string (32 bytes)',
    lowcardinality_string LowCardinality(String) COMMENT 'String with dictionary encoding',
    nullable_string Nullable(String) COMMENT 'Nullable string',

    -- Enum types
    enum8_value Enum8('option1' = 1, 'option2' = 2, 'option3' = 3) COMMENT 'Enum with 8-bit storage',
    enum16_value Enum16('value1' = 100, 'value2' = 200, 'value3' = 300) COMMENT 'Enum with 16-bit storage',

    -- Boolean (stored as UInt8)
    bool_value Bool COMMENT 'Boolean type (UInt8 internally)',

    -- Array types
    array_uint32 Array(UInt32) COMMENT 'Array of unsigned integers',
    array_string Array(String) COMMENT 'Array of strings',
    array_nullable Array(Nullable(UInt64)) COMMENT 'Array with nullable elements',

    -- Map types (tests Map filter generation)
    map_string_string Map(String, String) COMMENT 'String to string map',
    map_string_uint64 Map(String, UInt64) COMMENT 'String to uint64 map',

    -- Special types
    uuid_value UUID COMMENT 'UUID type',
    ipv4_value IPv4 COMMENT 'IPv4 address type',
    ipv6_value IPv6 COMMENT 'IPv6 address type',

    -- Nullable special types
    nullable_uuid Nullable(UUID) COMMENT 'Nullable UUID',
    nullable_ipv4 Nullable(IPv4) COMMENT 'Nullable IPv4'
)
ENGINE = MergeTree
PRIMARY KEY (id, timestamp)
ORDER BY (id, timestamp)
COMMENT 'Tests complex types: floats, decimals, strings, arrays, maps, enums, UUIDs, IPs';
