package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	colorGreen = "\033[0;32m"
	colorReset = "\033[0m"
)

func main() {
	input := flag.String("input", "", "Input file path")
	output := flag.String("output", "", "Output file path (defaults to input)")
	flag.Parse()

	if *input == "" {
		fmt.Println("Error: --input is required")
		os.Exit(1)
	}

	if *output == "" {
		*output = *input
	}

	content, err := os.ReadFile(*input)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Add ch tags to json tags
	processed := addChTags(string(content))

	// Fix array pointers (*[]T -> []T) for ClickHouse driver compatibility
	// oapi-codegen generates optional arrays as *[]T, but ClickHouse driver
	// cannot scan into pointer-to-slice, only into slice.
	processed = fixArrayPointers(processed)

	// Fix map pointers (*map[K]V -> map[K]V) for ClickHouse driver compatibility
	// oapi-codegen generates optional maps as *map[K]V, but ClickHouse driver
	// cannot scan into pointer-to-map, only into map.
	processed = fixMapPointers(processed)

	if err := os.WriteFile(*output, []byte(processed), 0600); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Post-processed %s%s\n", colorGreen, *output, colorReset)
}

// addChTags adds ch:"..." tags to struct fields that have json:"..." tags.
func addChTags(content string) string {
	// Pattern: `json:"field_name,omitempty"` or `json:"field_name"`
	// Replace with: `json:"field_name,omitempty" ch:"field_name"` or `json:"field_name" ch:"field_name"`

	// Match: `json:"something"` or `json:"something,omitempty"`
	re := regexp.MustCompile("`json:\"([^\"]+?)(?:,omitempty)?\"`")

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the field name from json tag.
		parts := strings.Split(match, `json:"`)
		if len(parts) < 2 {
			return match
		}

		jsonValue := strings.Split(parts[1], `"`)[0]
		// Remove ,omitempty if present.
		fieldName := strings.TrimSuffix(jsonValue, ",omitempty")

		// Add ch tag with the same field name (without omitempty).
		return strings.TrimSuffix(match, "`") + ` ch:"` + fieldName + `"` + "`"
	})

	return result
}

// fixArrayPointers converts *[]T to []T for ClickHouse driver compatibility.
//
// Why this is needed:
//   - oapi-codegen generates optional fields as pointers (e.g., *[]string)
//   - ClickHouse driver's Scan() cannot handle *[]T, only []T
//   - oapi-codegen has no configuration option to disable pointers for arrays only
func fixArrayPointers(content string) string {
	re := regexp.MustCompile(`\*\[\]([a-zA-Z0-9_]+)`)

	return re.ReplaceAllString(content, "[]$1")
}

// fixMapPointers converts *map[K]V to map[K]V for ClickHouse driver compatibility.
//
// Why this is needed:
//   - oapi-codegen generates optional maps as pointers (e.g., *map[string]string)
//   - ClickHouse driver's Scan() cannot handle *map[K]V, only map[K]V
//   - Maps are reference types in Go, nil map is already a valid zero value
func fixMapPointers(content string) string {
	re := regexp.MustCompile(`\*map\[([a-zA-Z0-9_]+)\]([a-zA-Z0-9_]+)`)

	return re.ReplaceAllString(content, "map[$1]$2")
}
