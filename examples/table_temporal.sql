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
