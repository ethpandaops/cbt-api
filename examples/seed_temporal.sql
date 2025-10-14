-- Seed data for fct_data_types_temporal (2 rows)
INSERT INTO fct_data_types_temporal (
    id,
    date_value, date32_value, datetime_value, datetime64_millis, datetime64_micros, datetime64_nanos,
    datetime_utc, datetime64_utc,
    nullable_date, nullable_datetime, nullable_datetime64
) VALUES
(
    1,
    '2024-01-15', '2024-01-15', '2024-01-15 10:30:45', '2024-01-15 10:30:45.123', '2024-01-15 10:30:45.123456', '2024-01-15 10:30:45.123456789',
    '2024-01-15 10:30:45', '2024-01-15 10:30:45.123',
    '2024-01-15', '2024-01-15 10:30:45', '2024-01-15 10:30:45.123'
),
(
    2,
    '2024-06-20', '2024-06-20', '2024-06-20 14:15:30', '2024-06-20 14:15:30.456', '2024-06-20 14:15:30.456789', '2024-06-20 14:15:30.456789012',
    '2024-06-20 14:15:30', '2024-06-20 14:15:30.456',
    NULL, NULL, NULL
);
