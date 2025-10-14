-- Seed data for fct_data_types_complex (2 rows)
INSERT INTO fct_data_types_complex (
    id, timestamp,
    float32_value, float64_value, nullable_float64,
    decimal32_value, decimal64_value, decimal128_value, nullable_decimal,
    string_value, fixed_string_value, lowcardinality_string, nullable_string,
    enum8_value, enum16_value,
    bool_value,
    array_uint32, array_string, array_nullable,
    map_string_string, map_string_uint64,
    uuid_value, ipv4_value, ipv6_value,
    nullable_uuid, nullable_ipv4
) VALUES
(
    1, 1699999000,
    3.14159, 2.718281828459045, 1.41421,
    1234.5678, 12345678.90123456, 123456789012345678.901234567890123456, 99.99,
    'test string 1', '0123456789abcdef0123456789abcdef', 'category_a', 'nullable text',
    'option1', 'value1',
    true,
    [1, 2, 3, 4, 5], ['apple', 'banana', 'cherry'], [100, NULL, 300],
    {'key1': 'value1', 'key2': 'value2'}, {'metric1': 100, 'metric2': 200},
    '550e8400-e29b-41d4-a716-446655440000', '192.168.1.1', '2001:0db8:85a3:0000:0000:8a2e:0370:7334',
    '550e8400-e29b-41d4-a716-446655440001', '10.0.0.1'
),
(
    2, 1699999100,
    1.61803, 3.141592653589793, NULL,
    9876.5432, 98765432.10987654, 987654321098765432.109876543210987654, NULL,
    'test string 2', 'fedcba9876543210fedcba9876543210', 'category_b', NULL,
    'option2', 'value2',
    false,
    [10, 20, 30], ['dog', 'elephant', 'fox'], [NULL, 200, NULL],
    {'foo': 'bar', 'baz': 'qux'}, {'score': 500, 'count': 1000},
    '660e8400-e29b-41d4-a716-446655440000', '172.16.0.1', '2001:0db8:85a3::8a2e:0370:7335',
    NULL, NULL
);
