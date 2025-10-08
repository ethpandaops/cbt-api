package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Parse flags
	openapiPath := flag.String("openapi", "openapi.yaml", "Path to OpenAPI spec")
	protoPath := flag.String("proto-path", ".xatu-cbt/pkg/proto/clickhouse",
		"Path to proto files")
	output := flag.String("output", "internal/server/implementation.go", "Output file")
	flag.Parse()

	fmt.Printf("Generating implementation from:\n")
	fmt.Printf("  OpenAPI: %s\n", *openapiPath)
	fmt.Printf("  Protos:  %s\n", *protoPath)
	fmt.Printf("  Output:  %s\n", *output)

	// 1. Load OpenAPI spec
	spec, err := loadOpenAPI(*openapiPath)
	if err != nil {
		fmt.Printf("Error loading OpenAPI: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded OpenAPI spec: %d endpoints\n", len(spec.Endpoints))

	// 2. Analyze proto files
	protoInfo, err := analyzeProtos(*protoPath)
	if err != nil {
		fmt.Printf("Error analyzing protos: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analyzed protos: %d filter types, %d query builders, %d request field mappings\n",
		len(protoInfo.FilterTypes), len(protoInfo.QueryBuilders), len(protoInfo.RequestFields))

	// 3. Generate code
	generator := &CodeGenerator{
		spec:      spec,
		protoInfo: protoInfo,
	}

	code := generator.Generate()

	// 4. Write output
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, []byte(code), 0600); err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	lines := len(strings.Split(code, "\n"))
	fmt.Printf("✓ Generated %d lines: %s\n", lines, *output)
}
