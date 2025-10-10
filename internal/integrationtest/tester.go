package integrationtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TestResult holds the result of testing a single endpoint.
type TestResult struct {
	Endpoint   Endpoint
	StatusCode int
	Success    bool
	ErrorBody  string
	Duration   time.Duration
}

// EndpointTester tests API endpoints using seed data to populate parameters.
type EndpointTester struct {
	BaseURL    string
	HTTPClient *http.Client
	SeedData   map[string][]map[string]any
}

// NewEndpointTester creates a new endpoint tester with the provided base URL
// and seed data. It initializes an HTTP client with a 30-second timeout.
func NewEndpointTester(
	baseURL string,
	seedData map[string][]map[string]any,
) *EndpointTester {
	return &EndpointTester{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		SeedData: seedData,
	}
}

// TestEndpoint tests a single endpoint by building a request URL with required
// parameters from seed data, executing the request, and returning the result.
// It measures the duration of the request and reads the error body for non-200
// responses.
func (t *EndpointTester) TestEndpoint(
	ctx context.Context,
	endpoint Endpoint,
) TestResult {
	result := TestResult{
		Endpoint: endpoint,
		Success:  false,
	}

	// Build request URL with parameters from seed data
	reqURL, err := t.buildRequestURL(endpoint)
	if err != nil {
		result.ErrorBody = fmt.Sprintf("failed to build request URL: %v", err)

		return result
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, reqURL, nil)
	if err != nil {
		result.ErrorBody = fmt.Sprintf("failed to create request: %v", err)

		return result
	}

	// Execute request and measure duration
	start := time.Now()
	resp, err := t.HTTPClient.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.ErrorBody = fmt.Sprintf("request failed: %v", err)

		return result
	}

	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read error body for non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result.ErrorBody = fmt.Sprintf(
				"status %d, failed to read error body: %v",
				resp.StatusCode,
				err,
			)
		} else {
			result.ErrorBody = string(body)
		}

		return result
	}

	result.Success = true

	return result
}

// TestAllEndpoints tests all provided endpoints sequentially and returns a
// slice of test results. Each endpoint is tested using the TestEndpoint method.
func (t *EndpointTester) TestAllEndpoints(
	ctx context.Context,
	endpoints []Endpoint,
) []TestResult {
	results := make([]TestResult, 0, len(endpoints))

	for _, endpoint := range endpoints {
		result := t.TestEndpoint(ctx, endpoint)
		results = append(results, result)
	}

	return results
}

// buildRequestURL constructs the full request URL with path parameters
// substituted and query parameters appended from seed data.
func (t *EndpointTester) buildRequestURL(endpoint Endpoint) (string, error) {
	// Start with base URL and path
	fullURL := t.BaseURL + endpoint.Path

	// Handle path parameters (e.g., /api/v1/table/{id})
	if strings.Contains(fullURL, "{") {
		for _, paramName := range endpoint.RequiredParams {
			placeholder := fmt.Sprintf("{%s}", paramName)
			if strings.Contains(fullURL, placeholder) {
				value, err := t.getParamValue(endpoint.TableName, paramName)
				if err != nil {
					return "", fmt.Errorf(
						"failed to get path param %s: %w",
						paramName,
						err,
					)
				}

				// URL-encode path parameters to handle special characters like /
				encodedValue := url.PathEscape(value)
				fullURL = strings.Replace(fullURL, placeholder, encodedValue, 1)
			}
		}
	}

	// Parse URL to add query parameters
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	query := parsedURL.Query()

	// Add explicitly required parameters
	for _, paramName := range endpoint.RequiredParams {
		// Skip path parameters (already handled)
		if strings.Contains(endpoint.Path, fmt.Sprintf("{%s}", paramName)) {
			continue
		}

		value, err := t.getParamValue(endpoint.TableName, paramName)
		if err != nil {
			return "", fmt.Errorf(
				"failed to get query param %s: %w",
				paramName,
				err,
			)
		}

		query.Add(paramName, value)
	}

	// Handle required groups (e.g., primary_key group with alternatives)
	// Add one parameter from each required group
	for groupName, paramNames := range endpoint.RequiredGroupParams {
		// Try to add the first available parameter from this group
		added := false

		for _, paramName := range paramNames {
			value, err := t.getParamValue(endpoint.TableName, paramName)
			if err == nil && value != "" {
				query.Add(paramName, value)

				added = true

				break // Only need one from each group
			}
		}

		if !added {
			// Couldn't add any parameter from this required group
			return "", fmt.Errorf(
				"could not satisfy required group %s: no valid parameters from %v",
				groupName,
				paramNames,
			)
		}
	}

	// For LIST endpoints with no required params and no required groups, add a primary key filter
	// This is needed because the API requires at least one primary key field
	if len(endpoint.RequiredParams) == 0 && len(endpoint.RequiredGroupParams) == 0 && !strings.Contains(endpoint.Path, "{") {
		if err := t.addPrimaryKeyFilter(endpoint.TableName, &query); err != nil {
			return "", fmt.Errorf(
				"failed to add primary key filter: %w",
				err,
			)
		}
	}

	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// getParamValue extracts a parameter value from the seed data for the given
// table. It handles filter suffixes (_gte, _lte, _eq):
// - _gte: returns the minimum value (first row)
// - _lte: returns the maximum value (last row)
// - _eq: returns an exact value (first row)
// - no suffix: returns the value from the first row.
func (t *EndpointTester) getParamValue(
	tableName string,
	paramName string,
) (string, error) {
	// Get the table's seed data
	rows, exists := t.SeedData[tableName]
	if !exists || len(rows) == 0 {
		return "", fmt.Errorf("no seed data for table %s", tableName)
	}

	// Determine the field name by stripping filter suffixes
	fieldName := paramName
	filterType := ""

	if strings.HasSuffix(paramName, "_gte") {
		fieldName = strings.TrimSuffix(paramName, "_gte")
		filterType = "gte"
	} else if strings.HasSuffix(paramName, "_lte") {
		fieldName = strings.TrimSuffix(paramName, "_lte")
		filterType = "lte"
	} else if strings.HasSuffix(paramName, "_eq") {
		fieldName = strings.TrimSuffix(paramName, "_eq")
		filterType = "eq"
	}

	// Get value based on filter type
	switch filterType {
	case "gte":
		// Use min value (first row)
		value, exists := rows[0][fieldName]
		if !exists {
			return "", fmt.Errorf(
				"field %s not found in seed data for table %s",
				fieldName,
				tableName,
			)
		}

		return formatParamValue(fieldName, value), nil

	case "lte":
		// Use max value (last row)
		value, exists := rows[len(rows)-1][fieldName]
		if !exists {
			return "", fmt.Errorf(
				"field %s not found in seed data for table %s",
				fieldName,
				tableName,
			)
		}

		return formatParamValue(fieldName, value), nil

	case "eq":
		// Use exact value (first row)
		value, exists := rows[0][fieldName]
		if !exists {
			return "", fmt.Errorf(
				"field %s not found in seed data for table %s",
				fieldName,
				tableName,
			)
		}

		return formatParamValue(fieldName, value), nil

	default:
		// No filter suffix, use value from first row
		value, exists := rows[0][paramName]
		if !exists {
			return "", fmt.Errorf(
				"field %s not found in seed data for table %s",
				paramName,
				tableName,
			)
		}

		return formatParamValue(paramName, value), nil
	}
}

// addPrimaryKeyFilter adds a primary key filter to the query for LIST endpoints.
// It looks for common primary key fields in the seed data and adds a _gte filter.
func (t *EndpointTester) addPrimaryKeyFilter(
	tableName string,
	query *url.Values,
) error {
	// Get the table's seed data
	rows, exists := t.SeedData[tableName]
	if !exists || len(rows) == 0 {
		return fmt.Errorf("no seed data for table %s", tableName)
	}

	// Common primary key field names in priority order
	primaryKeyFields := []string{
		"slot",
		"slot_start_date_time",
		"block_root",
		"chunk_start_block_number",
		"updated_date_time",
		"rank",
		"meta_client_name",
	}

	// Try to find a primary key field in the seed data
	firstRow := rows[0]
	for _, fieldName := range primaryKeyFields {
		if value, exists := firstRow[fieldName]; exists {
			// Add a _gte filter for this field
			paramName := fieldName + "_gte"
			paramValue := formatParamValue(fieldName, value)
			query.Add(paramName, paramValue)

			return nil
		}
	}

	// If no known primary key field found, use the first field
	for fieldName, value := range firstRow {
		paramName := fieldName + "_gte"
		paramValue := formatParamValue(fieldName, value)
		query.Add(paramName, paramValue)

		return nil
	}

	return fmt.Errorf("no fields found in seed data for table %s", tableName)
}

// formatParamValue formats a parameter value for use in URL.
// It handles special cases like DateTime values that need to be converted to Unix timestamps.
func formatParamValue(fieldName string, value any) string {
	// Check if this is a DateTime field
	if isDateTimeField(fieldName) {
		// Try to convert to Unix timestamp
		if timestamp := convertToUnixTimestamp(value); timestamp != "" {
			return timestamp
		}
	}

	// Handle numeric values without scientific notation
	switch v := value.(type) {
	case float64:
		// Check if it's actually an integer (no fractional part)
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}

		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		if v == float32(int32(v)) {
			return strconv.FormatInt(int64(v), 10)
		}

		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	}

	// For non-DateTime fields or if conversion fails, use string representation
	return fmt.Sprintf("%v", value)
}

// isDateTimeField checks if a field name indicates a DateTime column.
func isDateTimeField(fieldName string) bool {
	return strings.Contains(strings.ToLower(fieldName), "date_time") ||
		strings.Contains(strings.ToLower(fieldName), "datetime") ||
		strings.Contains(strings.ToLower(fieldName), "timestamp")
}

// convertToUnixTimestamp converts a DateTime value to Unix timestamp string.
func convertToUnixTimestamp(value any) string {
	// Handle string DateTime values from ClickHouse JSON
	if strVal, ok := value.(string); ok {
		// Try common DateTime formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05.000",
			"2006-01-02T15:04:05.000",
			time.RFC3339,
		}

		for _, format := range formats {
			if t, err := time.Parse(format, strVal); err == nil {
				return strconv.FormatInt(t.Unix(), 10)
			}
		}
	}

	// Handle numeric values (already timestamps)
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	}

	// If we can't convert, return empty string
	return ""
}
