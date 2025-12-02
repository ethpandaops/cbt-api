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

    -- Array types (all variants to ensure complete filter type generation)
    array_int32 Array(Int32) COMMENT 'Array of signed 32-bit integers',
    array_int64 Array(Int64) COMMENT 'Array of signed 64-bit integers',
    array_uint32 Array(UInt32) COMMENT 'Array of unsigned 32-bit integers',
    array_uint64 Array(UInt64) COMMENT 'Array of unsigned 64-bit integers',
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
