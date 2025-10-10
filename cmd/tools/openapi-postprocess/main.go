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

	// Add ch tags to json tags.
	processed := addChTags(string(content))

	// Fix array pointers (*[]T -> []T) for ClickHouse compatibility
	processed = fixArrayPointers(processed)

	if err := os.WriteFile(*output, []byte(processed), 0600); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Added ch tags to %s%s\n", colorGreen, *output, colorReset)
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

// fixArrayPointers removes pointer indirection from array/slice types for ClickHouse compatibility.
// ClickHouse driver cannot scan Array columns into pointer-to-slice (*[]T), only into slices ([]T).
func fixArrayPointers(content string) string {
	// Match: *[]Type (pointer to slice)
	// Replace with: []Type (slice)
	re := regexp.MustCompile(`\*\[\]([a-zA-Z0-9_]+)`)

	return re.ReplaceAllString(content, "[]$1")
}
