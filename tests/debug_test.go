package client_test

import (
	"github.com/tinywasm/client"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDebugWasmExecGeneration helps debug the wasm_exec.js generation issue
func TestDebugWasmExecGeneration(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create a .wasm.go file
	wasmGoPath := filepath.Join(testDir, "module.wasm.go")
	content := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(wasmGoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test .wasm.go file: %v", err)
	}

	// Create client.WasmClient instance with verbose logging
	messages := []string{}
	config := &client.Config{
		SourceDir: func() string { return "web" },
		OutputDir: func() string { return "theme/js" },
	}

	tinyWasm := client.New(config)
	tinyWasm.SetLog(func(message ...any) {
		msg := fmt.Sprint(message...)
		messages = append(messages, msg)
		t.Log("LOG:", msg)
	})
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetEnableWasmExecJsOutput(true)
	tinyWasm.SetWasmExecJsOutputDir("theme/js")

	// Print all debug messages
	t.Log("=== All debug messages ===")
	for i, msg := range messages {
		t.Logf("%d: %s", i, msg)
	}

	// Check if wasm_exec.js was created
	wasmExecPath := filepath.Join(testDir, config.OutputDir(), "wasm_exec.js")
	t.Logf("Checking path: %s", wasmExecPath)

	info, err := os.Stat(wasmExecPath)
	if err != nil {
		t.Fatalf("wasm_exec.js not created: %v", err)
	}

	t.Logf("File size: %d bytes", info.Size())

	// Read and client.Log the content
	data, err := os.ReadFile(wasmExecPath)
	if err != nil {
		t.Fatalf("Failed to read wasm_exec.js: %v", err)
	}

	content = string(data)
	t.Logf("Content length: %d", len(content))
	t.Logf("Content preview (first 500 chars):\n%s", content[:min(500, len(content))])

	t.Logf("=== client.WasmClient State ===")
	t.Logf("client.TinyGoCompilerFlag: %v", tinyWasm.TinyGoCompilerFlag)
	t.Logf("client.TinyGoInstalled: %v", tinyWasm.TinyGoInstalled)
	t.Logf("client.CurrentSizeMode: %s", tinyWasm.CurrentSizeMode)

	// Check WasmProjectTinyGoJsUse
	isWasm, useTinyGo := tinyWasm.WasmProjectTinyGoJsUse()
	t.Logf("WasmProjectTinyGoJsUse: wasmProject=%v, useTinyGo=%v", isWasm, useTinyGo)

	// Check client.RequiresTinyGo for current mode
	requiresTiny := tinyWasm.RequiresTinyGo(tinyWasm.CurrentSizeMode)
	t.Logf("client.RequiresTinyGo(%s): %v", tinyWasm.CurrentSizeMode, requiresTiny)

	// Check for Go signatures
	goSignatures := client.WasmExecGoSignatures()
	t.Logf("Checking for %d Go signatures: %v", len(goSignatures), goSignatures)

	for _, signature := range goSignatures {
		found := strings.Contains(content, signature)
		t.Logf("Signature '%s': %v", signature, found)
	}

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
