//go:build tools
// +build tools

// Package tools anchors dependencies that are only referenced by generated
// code, so `go mod tidy` does not remove them from go.mod.
//
// oapi-codegen produces internal/handlers/generated.go (which is gitignored and
// imports github.com/oapi-codegen/runtime). Because that file is not committed,
// tidy cannot see the import on a clean checkout and would otherwise drop the
// module from go.mod, breaking the release build at compile time.
package tools

import (
	_ "github.com/oapi-codegen/runtime"
)
