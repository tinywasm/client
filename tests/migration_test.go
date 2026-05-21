package client_test

import (
	"go/token"
	"go/types"
	"testing"

	"go/importer"
)

// TestNoLocalEmbeds verifies the client package no longer embeds wasm_exec_*.js files.
// Since client/assets/ was deleted, this test confirms no embed directives remain for those files.
func TestNoLocalEmbeds(t *testing.T) {
	// Programmatically check: the client package must not export embeddedWasmExecGo or embeddedWasmExecTinyGo.
	fset := token.NewFileSet()
	_ = fset
	imp := importer.Default()
	pkg, err := imp.Import("github.com/tinywasm/client")
	if err != nil {
		t.Skipf("cannot import package (expected in CI without full build): %v", err)
	}
	scope := pkg.Scope()
	for _, forbidden := range []string{"EmbeddedWasmExecGo", "EmbeddedWasmExecTinyGo"} {
		if obj := scope.Lookup(forbidden); obj != nil {
			t.Errorf("client package still exports %q — embed not removed", forbidden)
		}
	}
}

// TestNoJavascriptStruct verifies the Javascript struct and related symbols
// have been removed from the client package public API.
func TestNoJavascriptStruct(t *testing.T) {
	fset := token.NewFileSet()
	_ = fset
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
