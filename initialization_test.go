package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitialization verifies basic WasmClient initialization
func TestInitialization(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "web/public",
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetBuildShortcuts("L", "M", "S")

	if tinyWasm.currenSizeMode != "L" {
		t.Errorf("Expected currenSizeMode to be L, got %s", tinyWasm.currenSizeMode)
	}
}

// TestWasmExecJsGeneration tests that the wasm_exec.js
// initialization file is created in the configured output directory when enabled.
func TestWasmExecJsGeneration(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetEnableWasmExecJsOutput(true)
	tinyWasm.SetWasmExecJsOutputDir("theme/js")

	// Ensure wasm_exec.js was created in the output path and is non-empty
	wasmExecPath := filepath.Join(testDir, "theme/js", "wasm_exec.js")
	info, err := os.Stat(wasmExecPath)
	if err != nil {
		t.Fatalf("Expected wasm_exec.js to be created at %s: %v", wasmExecPath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("Expected wasm_exec.js at %s to be non-empty", wasmExecPath)
	}

	// Verify the generated wasm_exec.js contains Go signatures (default mode is L)
	data, err := os.ReadFile(wasmExecPath)
	if err != nil {
		t.Fatalf("Failed to read generated wasm_exec.js: %v", err)
	}
	content := string(data)
	goSignatures := wasm_execGoSignatures()
	found := false
	for _, s := range goSignatures {
		if strings.Contains(content, s) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected generated wasm_exec.js to include Go signatures, none found")
	}
}

// TestDefaultConfiguration tests reaching the correct path for wasm_exec.js
func TestDefaultConfiguration(t *testing.T) {
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir("/test")
	tinyWasm.SetBuildShortcuts("c", "d", "p")

	expected := "web/js"
	tinyWasm.SetWasmExecJsOutputDir(expected)
	if !strings.Contains(tinyWasm.WasmExecJsOutputPath(), expected) {
		t.Errorf("Expected WasmExecJsOutputPath to contain %s, got %s", expected, tinyWasm.WasmExecJsOutputPath())
	}
}

// TestCreateDefaultFile tests creating the default WASM file
func TestCreateDefaultFile(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)

	// Now create the default WASM file
	result := tinyWasm.CreateDefaultWasmFileClientIfNotExist()
	if result == nil {
		t.Error("Expected CreateDefaultWasmFileClientIfNotExist to return WasmClient instance")
	}

	// Verify the default file was created
	expectedPath := filepath.Join(testDir, "web", "client.go")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected default WASM file to be created at %s", expectedPath)
	}
}
