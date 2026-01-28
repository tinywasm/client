package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCompileAllModes attempts to compile the WASM main file to disk
// using the three supported modes: fast (go), debugging (tinygo), minimal (tinygo).
// Simulates the real integration flow: InitialRegistration -> NewFileEvent -> Change modes.
// If tinygo is not present in PATH, the tinygo modes are skipped.
func TestCompileAllModes(t *testing.T) {
	// Create isolated temp workspace
	tmp := t.TempDir()
	webDirName := "web"
	webDir := filepath.Join(tmp, webDirName)
	publicDir := filepath.Join(webDir, "public")
	jsDir := filepath.Join(webDir, "theme", "js")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("failed to create test dirs: %v", err)
	}
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		t.Fatalf("failed to create test dirs: %v", err)
	}

	// Write a minimal go.mod
	goModContent := `module test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a minimal main.go
	mainWasmPath := filepath.Join(webDir, "client.go")
	wasmContent := `package main

func main() {
    println("hello wasm")
}
`
	if err := os.WriteFile(mainWasmPath, []byte(wasmContent), 0644); err != nil {
		t.Fatalf("failed to write client.go: %v", err)
	}

	// Create OutputDir to avoid chdir errors
	if err := os.MkdirAll(filepath.Join(tmp, webDirName, "public"), 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Prepare config with logger to prevent nil pointer dereference
	var logMessages []string
	cfg := NewConfig()
	w := New(cfg) // Initialize w first to access w.Config
	w.Config.SourceDir = func() string { return webDirName }
	w.Config.OutputDir = func() string { return filepath.Join(webDirName, "public") }
	cfg.Database = &testDatabase{data: make(map[string]string)}

	w.SetLog(func(message ...any) {
		logMessages = append(logMessages, fmt.Sprint(message...))
	})
	w.SetAppRootDir(tmp)
	w.SetWasmExecJsOutputDir(filepath.Join(webDirName, "theme", "js"))
	// Force External storage for this test as it verifies disk artifacts
	w.storage = &diskStorage{client: w}
	// Allow tests to enable tinygo detection by setting the private field
	w.tinyGoCompiler = true

	// Debug: Check initial state
	if w.Value() != "L" {
		t.Fatalf("Initial mode should be 'L', got '%s'", w.Value())
	}

	// Check tinygo availability
	_, err := exec.LookPath("tinygo")
	tinygoPresent := err == nil

	// Step 1: Simulate InitialRegistration flow - notify about existing file
	err = w.NewFileEvent("client.go", ".go", mainWasmPath, "create")
	if err != nil {
		t.Fatalf("NewFileEvent with create event failed: %v", err)
	}

	outPath := func() string {
		return filepath.Join(tmp, w.Config.OutputDir(), "client.wasm")
	}

	// Initial compile in coding mode to get a baseline file size
	fi, err := os.Stat(outPath())
	if err != nil {
		t.Fatalf("coding mode: expected output file at %s, got error: %v", outPath(), err)
	}
	codingModeFileSize := fi.Size()
	if codingModeFileSize == 0 {
		t.Fatalf("coding mode: output file exists but is empty: %s", outPath())
	}

	// Test JavaScript generation for initial coding mode (Go compiler)
	goJS, err := w.GetSSRClientInitJS()
	if err != nil {
		t.Errorf("coding mode: GetSSRClientInitJS failed: %v", err)
	} else {
		if len(goJS) == 0 {
			t.Errorf("coding mode: GetSSRClientInitJS returned empty JavaScript")
		}
	}

	// Test cases for mode switching
	tests := []struct {
		mode         string
		name         string
		requiresTiny bool
		assertSize   func(t *testing.T, size int64)
	}{
		{
			mode: "M", name: "debugging", requiresTiny: true,
			assertSize: func(t *testing.T, size int64) {
				if size == codingModeFileSize {
					t.Errorf("debugging mode file size (%d) should be different from coding mode size (%d)", size, codingModeFileSize)
				}
			},
		},
		{
			mode: "S", name: "production", requiresTiny: true,
			assertSize: func(t *testing.T, size int64) {
				if size == codingModeFileSize {
					t.Errorf("production mode file size (%d) should be different from coding mode size (%d)", size, codingModeFileSize)
				}
				// Production should be smaller than debug, but let's check against coding for simplicity
				if size >= codingModeFileSize {
					t.Errorf("production mode file size (%d) should be smaller than coding mode size (%d)", size, codingModeFileSize)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.requiresTiny && !tinygoPresent {
				t.Skipf("tinygo not in PATH; skipping %s mode", tc.name)
			}

			// Clear all JavaScript caches before each subtest to ensure clean state
			w.ClearJavaScriptCache()

			// Step 2: Change compilation mode
			w.Change(tc.mode)

			// Assert that the internal mode has changed
			if w.Value() != tc.mode {
				t.Fatalf("After Change, expected mode '%s', got '%s'", tc.mode, w.Value())
			}

			// CRITICAL: Verify that the mode is saved in the Database
			saved, err := cfg.Database.Get(StoreKeySizeMode)
			if err != nil {
				t.Errorf("Failed to get mode from store for %s: %v", tc.name, err)
			} else if saved != tc.mode {
				t.Errorf("Mode %s: Store mismatch. Expected: '%s', got: '%s'", tc.name, tc.mode, saved)
			}

			// Test JavaScript generation after mode change
			modeJS, err := w.GetSSRClientInitJS()
			if err != nil {
				t.Errorf("%s mode: GetSSRClientInitJS failed: %v", tc.name, err)
				return
			}

			if len(modeJS) == 0 {
				t.Errorf("%s mode: GetSSRClientInitJS returned empty JavaScript", tc.name)
				return
			}

			// Clear cache to test fresh generation
			w.ClearJavaScriptCache()

			// Test again to verify cache clearing works
			freshJS, freshErr := w.GetSSRClientInitJS()
			if freshErr != nil {
				t.Errorf("%s mode: GetSSRClientInitJS after cache clear failed: %v", tc.name, freshErr)
			} else if modeJS != freshJS {
				t.Errorf("%s mode: JavaScript differs after cache clear (length %d vs %d)", tc.name, len(modeJS), len(freshJS))
			}

			// Step 3: Simulate file modification event to trigger re-compilation
			err = w.NewFileEvent("main.go", ".go", mainWasmPath, "write")
			if err != nil {
				t.Fatalf("mode %s: NewFileEvent with write event failed: %v", tc.name, err)
			}

			// Step 4: Verify output file and its size
			fi, err := os.Stat(outPath())
			if err != nil {
				t.Fatalf("mode %s: expected output file at %s, got error: %v", tc.name, outPath(), err)
			}

			// Use the specific assertion for the test case
			tc.assertSize(t, fi.Size())

		})
	}

	// Verify that Go and TinyGo generate different JavaScript
	if tinygoPresent {
		// Switch to a TinyGo mode to get TinyGo JavaScript
		w.Change("M")

		tinygoJS, err := w.GetSSRClientInitJS()
		if err != nil {
			t.Errorf("Failed to get TinyGo JavaScript: %v", err)
		} else if len(tinygoJS) > 0 && len(goJS) > 0 {
			if goJS == tinygoJS {
				t.Errorf("Go and TinyGo should generate different JavaScript but they are identical (lengths: Go=%d, TinyGo=%d)", len(goJS), len(tinygoJS))
			} else {
				t.Logf("SUCCESS: Go and TinyGo generate different JavaScript (lengths: Go=%d, TinyGo=%d)", len(goJS), len(tinygoJS))
			}
		}
	}
}
