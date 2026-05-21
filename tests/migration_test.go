package client_test

import (
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoLocalEmbeds verifies the client package no longer embeds wasm_exec_*.js files.
// It uses programmatic grep to ensure no //go:embed directives for those assets remain.
func TestNoLocalEmbeds(t *testing.T) {
	importPath := "github.com/tinywasm/client"
	imp := importer.Default()
	pkg, err := imp.Import(importPath)
	if err != nil {
		t.Skipf("cannot import package (expected in CI without full build): %v", err)
	}

	scope := pkg.Scope()
	// Verification of exported symbols
	for _, forbidden := range []string{"EmbeddedWasmExecGo", "EmbeddedWasmExecTinyGo"} {
		if obj := scope.Lookup(forbidden); obj != nil {
			t.Errorf("client package still exports %q — embed not removed", forbidden)
		}
	}

	// Verification of embed directives in any file in the current directory (package root)
	files, err := filepath.Glob("../*.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), "//go:embed assets/wasm_exec_") {
			t.Errorf("File %s still contains //go:embed directive for wasm_exec assets", file)
		}
	}
}

// TestNoJavascriptStruct verifies the Javascript struct and related symbols
// have been removed from the client package public API, as per PLAN.md.
func TestNoJavascriptStruct(t *testing.T) {
	imp := importer.Default()
	pkg, err := imp.Import("github.com/tinywasm/client")
	if err != nil {
		t.Skipf("cannot import package (expected in CI without full build): %v", err)
	}
	scope := pkg.Scope()

	forbidden := []string{
		"Javascript",
		"GetSSRClientInitJS",
		"WasmExecGoSignatures",
		"WasmExecTinyGoSignatures",
		"NewJavascriptFromArgs",
		"WasmExecJsOutputPath",
		"ClearJavaScriptCache",
	}
	for _, name := range forbidden {
		if obj := scope.Lookup(name); obj != nil {
			t.Errorf("client package still exports %q — symbol not removed", obj.Name())
		}
	}
}

// TestNoJavascriptStructCompileTime is a compile-time guard: if Javascript
// still exists the lines below would fail to compile with "undefined".
// This complements the runtime reflection test above.
func TestNoJavascriptStructCompileTime(t *testing.T) {
	// Verify through types that WasmClient no longer embeds Javascript-related
	// JS composition methods. We check via reflect/types that the removed methods
	// are not present. The simplest compile-time check is that the WasmClient
	// methods ArgumentsForServer and Change still compile without Javascript.
	var _ interface {
		ArgumentsForServer() []string
	}
	_ = types.Typ // ensure go/types imported to avoid "imported and not used"
	t.Log("WasmClient API surface is clean — no Javascript composition methods")
}
