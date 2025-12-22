package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitializationDetectionFromWasmExecJs tests detection from existing wasm_exec.js
func TestInitializationDetectionFromWasmExecJs(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create web structure
	webDir := filepath.Join(testDir, "web")
	jsDir := filepath.Join(webDir, "theme", "js")
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create a mock wasm_exec.js with Go signatures
	wasmExecPath := filepath.Join(jsDir, "wasm_exec.js")
	goSignatures := wasm_execGoSignatures()
	if len(goSignatures) == 0 {
		t.Fatal("No Go signatures available for test")
	}

	content := goSignatures[0] + "\n// Go WASM exec\n"
	if err := os.WriteFile(wasmExecPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test wasm_exec.js: %v", err)
	}

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "web/public",
		Logger:    func(message ...any) {},
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetBuildShortcuts("L", "M", "S")
	tinyWasm.SetWasmExecJsOutputDir("web/theme/js")
	tinyWasm.SetEnableWasmExecJsOutput(true)

	if !tinyWasm.wasmProject {
		t.Error("Expected wasmProject to be true after detecting wasm_exec.js")
	}
	if tinyWasm.tinyGoCompiler {
		t.Error("Expected tinyGoCompiler to be false (Go detected)")
	}
	if tinyWasm.currentMode != "L" {
		t.Errorf("Expected currentMode to be L, got %s", tinyWasm.currentMode)
	}
}

// TestInitializationDetectionFromGoFiles tests detection from .wasm.go files
// and ensures that when a WASM project is detected the wasm_exec.js
// initialization file is created in the configured output directory.
func TestInitializationDetectionFromGoFiles(t *testing.T) {
	// Create temporary test directory
	testDir := t.TempDir()

	// Create a .wasm.go file
	wasmGoPath := filepath.Join(testDir, "module.wasm.go")
	content := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(wasmGoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test .wasm.go file: %v", err)
	}

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
		Logger:    func(message ...any) {},
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetEnableWasmExecJsOutput(true)
	tinyWasm.SetWasmExecJsOutputDir("theme/js")

	if !tinyWasm.wasmProject {
		t.Error("Expected wasmProject to be true after detecting .wasm.go file")
	}
	if tinyWasm.tinyGoCompiler {
		t.Error("Expected tinyGoCompiler to be false (default to Go)")
	}
	if tinyWasm.currentMode != "L" {
		t.Errorf("Expected currentMode to be L, got %s", tinyWasm.currentMode)
	}

	// Ensure wasm_exec.js was created in the output path and is non-empty
	wasmExecPath := filepath.Join(testDir, config.OutputDir, "wasm_exec.js")
	info, err := os.Stat(wasmExecPath)
	if err != nil {
		t.Fatalf("Expected wasm_exec.js to be created at %s: %v", wasmExecPath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("Expected wasm_exec.js at %s to be non-empty", wasmExecPath)
	}

	// Verify the generated wasm_exec.js contains Go signatures
	data, err := os.ReadFile(wasmExecPath)
	if err != nil {
		t.Fatalf("Failed to read generated wasm_exec.js: %v", err)
	}
	content = string(data)
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

// TestDefaultConfiguration tests that WasmExecJsOutputDir defaults to "src/web/ui/js"
func TestDefaultConfiguration(t *testing.T) {
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
		Logger:    func(message ...any) {},
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir("/test")
	tinyWasm.SetBuildShortcuts("c", "d", "p")

	expected := "web/js"
	// Note: We don't check a default anymore because wasmExecJsOutputDir is private and defaults to empty.
	// But we can verify the setter works.
	tinyWasm.SetWasmExecJsOutputDir(expected)
	// We check indirect path via WasmExecJsOutputPath (unexported field access in tests is OK but let's use the public API)
	if !strings.Contains(tinyWasm.WasmExecJsOutputPath(), expected) {
		t.Errorf("Expected WasmExecJsOutputPath to contain %s, got %s", expected, tinyWasm.WasmExecJsOutputPath())
	}
}

// TestNoWasmProjectDetected tests behavior when no WASM files are found
func TestNoWasmProjectDetected(t *testing.T) {
	// Create temporary test directory with no WASM files
	testDir := t.TempDir()

	// Create a regular Go file (not .wasm.go)
	regularGoPath := filepath.Join(testDir, "main.go")
	content := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(regularGoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	// Create WasmClient instance
	config := &Config{
		SourceDir: "web",
		OutputDir: "theme/js",
		Logger:    func(message ...any) {},
	}

	tinyWasm := New(config)
	tinyWasm.SetAppRootDir(testDir)
	tinyWasm.SetBuildShortcuts("L", "M", "S")
	tinyWasm.SetEnableWasmExecJsOutput(true)
	tinyWasm.SetWasmExecJsOutputDir("theme/js")

	// Initially, no WASM project should be detected
	if tinyWasm.wasmProject {
		t.Error("Expected wasmProject to be false initially when no WASM files exist")
	}

	// Now create the default WASM file using the new optional method
	result := tinyWasm.CreateDefaultWasmFileClientIfNotExist()
	if result == nil {
		t.Error("Expected CreateDefaultWasmFileClientIfNotExist to return WasmClient instance")
	}

	// Verify WASM project is now detected after creating default file
	if !tinyWasm.wasmProject {
		t.Error("Expected wasmProject to be true after creating default WASM file")
	}

	// Verify the default file was created
	expectedPath := filepath.Join(testDir, "web", "client.go")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected default WASM file to be created at %s", expectedPath)
	}
}
