package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileEvent(t *testing.T) {
	// Setup test environment with an isolated temporary directory
	tmp := t.TempDir()
	// SourceDir should be the subfolder name under AppRootDir
	sourceDirName := "wasmTest"
	sourceDir := filepath.Join(tmp, sourceDirName)

	outputDir := filepath.Join(tmp, "output")
	// Create directories
	for _, dir := range []string{sourceDir, outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Error creating test directory: %v", err)
		}
	}

	// Write a minimal go.mod
	goModPath := filepath.Join(tmp, "go.mod")
	goModContent := `module test

go 1.21
`
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Configure WasmClient handler with a logger for testing output
	var logMessages []string
	config := &Config{
		SourceDir: func() string { return sourceDirName },
		OutputDir: func() string { return "output" },
	}

	tinyWasm := New(config)
	tinyWasm.SetLog(func(message ...any) {
		logMessages = append(logMessages, fmt.Sprint(message...))
	})
	tinyWasm.SetAppRootDir(tmp)
	// Force External storage for tests that expect files on disk
	tinyWasm.storage = &diskStorage{client: tinyWasm}
	t.Run("Verify client.go compilation", func(t *testing.T) {
		mainWasmPath := filepath.Join(tmp, sourceDirName, "client.go") // client.go in source root
		// defer os.Remove(mainWasmPath)  // Removed to allow subsequent tests

		// Create main wasm file
		content := `package main

		func main() {
			println("Hello WasmClient!")
		}`
		os.WriteFile(mainWasmPath, []byte(content), 0644)

		err := tinyWasm.NewFileEvent("client.go", ".go", mainWasmPath, "write")
		if err != nil {
			t.Fatal(err)
		}

		// Verify wasm file was created
		outputPath := tinyWasm.MainOutputFileAbsolutePath()
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("client.wasm file was not created")
		}
	})
	t.Run("Verify module wasm compilation now goes to main.wasm", func(t *testing.T) {
		// Create main.go in the web root first
		mainWasmPath := filepath.Join(tmp, sourceDirName, "main.go") // main.go in source root
		mainContent := `package main

		func main() {
			println("Main WASM entry point")
		}`
		os.WriteFile(mainWasmPath, []byte(mainContent), 0644)

		// Create another .wasm.go file in sourceDir to simulate additional WASM entry
		moduleWasmPath := filepath.Join(tmp, sourceDirName, "users.wasm.go")
		content := `package main

		func main() {
			println("Hello Users Module with WasmClient!")
		}`
		os.WriteFile(moduleWasmPath, []byte(content), 0644)

		err := tinyWasm.NewFileEvent("users.wasm.go", ".go", moduleWasmPath, "write")
		if err != nil {
			t.Fatal(err)
		}

		// Verify client.wasm file was created (single output)
		outputPath := tinyWasm.MainOutputFileAbsolutePath()
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("client.wasm file was not created")
		}
		// Individual per-module wasm outputs are deprecated; ensure main output exists
		oldOutputPath := tinyWasm.MainOutputFileAbsolutePath()
		if _, err := os.Stat(oldOutputPath); os.IsNotExist(err) {
			t.Fatal("client.wasm file was not created")
		}
	})

	t.Run("Handle invalid file path", func(t *testing.T) {
		err := tinyWasm.NewFileEvent("invalid.go", ".go", "", "write")
		if err == nil {
			t.Fatal("Expected error for invalid file path")
		}
	})

	t.Run("Handle non-write event", func(t *testing.T) {
		mainWasmPath := filepath.Join(tmp, sourceDirName, "main.wasm.go")
		err := tinyWasm.NewFileEvent("main.wasm.go", ".go", mainWasmPath, "remove")
		if err != nil {
			t.Fatal("Expected no error for non-write event")
		}
	})
	t.Run("Verify TinyGo compiler is configurable", func(t *testing.T) {
		// Test initial configuration
		var logMessages []string
		config := NewConfig()
		config.SourceDir = func() string { return sourceDirName }
		config.OutputDir = func() string { return "output" }

		tinyWasm := New(config)
		tinyWasm.SetLog(func(message ...any) {
			logMessages = append(logMessages, fmt.Sprint(message...))
		})
		tinyWasm.SetAppRootDir(tmp)
		// Tests run inside the package; set private tinyGoCompiler explicitly
		tinyWasm.tinyGoCompiler = false // Start with Go standard compiler

		// Verify initial state (should be coding mode)
		if tinyWasm.Value() != "L" {
			t.Fatal("Expected coding mode to be used initially")
		}

		// Test setting TinyGo compiler (debug mode)
		var changeMsg string
		tinyWasm.SetLog(func(message ...any) {
			if len(message) > 0 {
				changeMsg = fmt.Sprint(message...)
			}
		})
		tinyWasm.Change("M")

		// If TinyGo isn't available, log likely contains an error message
		if strings.Contains(strings.ToLower(changeMsg), "cannot") || strings.Contains(strings.ToLower(changeMsg), "not available") {
			t.Logf("TinyGo not available: %s", changeMsg)
		} else {
			// Check that we successfully switched to Medium mode (debug)
			if tinyWasm.Value() != "M" {
				t.Fatal("Expected Medium mode (debug) to be set after change")
			}
			// Message can be success, warning, or auto-compilation error (in test env)
			// Accept "Medium" (new format) or "debug" (legacy) or "warning" or "error"
			msgLower := strings.ToLower(changeMsg)
			if !strings.Contains(msgLower, "medium") && !strings.Contains(msgLower, "debug") &&
				!strings.Contains(msgLower, "warning") && !strings.Contains(msgLower, "error") &&
				!strings.Contains(msgLower, "wasm") {
				t.Fatalf("Expected Medium mode message, WASM success, warning, or error, got: %s", changeMsg)
			}
		}
	})
}

// Test for UnobservedFiles method
func TestUnobservedFiles(t *testing.T) {
	tmp := t.TempDir()
	var logMessages []string
	config := &Config{
		SourceDir: func() string { return "web" },
		OutputDir: func() string { return "public" },
	}

	tinyWasm := New(config)
	tinyWasm.SetLog(func(message ...any) {
		logMessages = append(logMessages, fmt.Sprint(message...))
	})
	tinyWasm.SetAppRootDir(tmp)
	unobservedFiles := tinyWasm.UnobservedFiles()
	// Should contain client.wasm and client_temp.wasm (generated files from gobuild)
	expectedFiles := []string{"client.wasm", "client_temp.wasm"}
	if len(unobservedFiles) != len(expectedFiles) {
		t.Logf("Actual unobserved files: %v", unobservedFiles)
		t.Logf("Expected unobserved files: %v", expectedFiles)
		t.Fatalf("Expected %d unobserved files, got %d", len(expectedFiles), len(unobservedFiles))
	}

	for i, expected := range expectedFiles {
		if unobservedFiles[i] != expected {
			t.Errorf("Expected unobserved file %q, got %q", expected, unobservedFiles[i])
		}
	}

	// Verify client.go is NOT in unobserved files (should be watched)
	for _, file := range unobservedFiles {
		if file == "client.go" {
			t.Error("client.go should NOT be in unobserved files - it should be watched for changes")
		}
	}
}
