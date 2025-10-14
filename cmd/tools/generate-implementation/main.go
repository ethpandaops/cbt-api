package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethpandaops/xatu-cbt-api/internal/config"
)

const (
	colorGreen = "\033[0;32m"
	colorReset = "\033[0m"
)

func main() {
	// Parse flags
	openapiPath := flag.String("openapi", "openapi.yaml", "Path to OpenAPI spec")
	protoPath := flag.String("proto-path", ".xatu-cbt/pkg/proto/clickhouse",
		"Path to proto files")
	output := flag.String("output", "internal/server/implementation.go", "Output file")
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	basePath := flag.String("base-path", "/api/v1", "API base path (defaults to /api/v1, overridden by config if present)")
	flag.Parse()

	// 0. Load config to get api.base_path (optional - falls back to flag default)
	apiBasePath := *basePath

	if _, err := os.Stat(*configFile); err == nil {
		cfg, err := config.Load(*configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		apiBasePath = cfg.API.BasePath
	}

	// 1. Load OpenAPI spec
	spec, err := loadOpenAPI(apiBasePath, *openapiPath)
	if err != nil {
		fmt.Printf("Error loading OpenAPI: %v\n", err)
		os.Exit(1)
	}

	// 2. Analyze proto files
	protoInfo, err := analyzeProtos(*protoPath)
	if err != nil {
		fmt.Printf("Error analyzing protos: %v\n", err)
		os.Exit(1)
	}

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
	fmt.Printf("%s✓ Generated %d lines: %s%s\n", colorGreen, lines, *output, colorReset)
}
