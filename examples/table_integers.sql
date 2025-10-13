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
