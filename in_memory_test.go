package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInMemoryRefactoring(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock source file (so we can compile)
	srcDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "client.go")
	// Simple valid Go program
	err := os.WriteFile(srcFile, []byte(`package main
import "fmt"
func main() { fmt.Println("WASM") }`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:       "web",
		OutputDir:       "web/public",
		AssetsURLPrefix: "assets",
		Logger: func(msg ...any) {
			t.Log(msg...)
		},
	}

	// 1. Initialize - Should be In-Memory (no .wasm file yet)
	c := New(cfg)
	c.SetAppRootDir(tmpDir)
	c.SetMainInputFile("client.go")
	c.SetOutputName("test-client")

	if c.strategy.Name() != "In-Memory" {
		t.Errorf("Expected In-Memory strategy, got %s", c.strategy.Name())
	}

	// 2. Trigger Event -> Compile to Memory
	// This should populate the internal buffer
	err = c.NewFileEvent("client.go", ".go", srcFile, "write")
	if err != nil {
		t.Fatalf("NewFileEvent failed: %v", err)
	}

	// 3. Register Routes and Verify Serving
	mux := http.NewServeMux()
	c.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Expected path with prefix
	url := ts.URL + "/assets/test-client.wasm"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/wasm" {
		t.Errorf("Expected Content-Type application/wasm, got %s", ct)
	}

	// Verify file was NOT written to disk
	diskPath := filepath.Join(tmpDir, "web/public/test-client.wasm")
	if _, err := os.Stat(diskPath); err == nil {
		t.Error("Expected no file on disk in In-Memory mode")
	}

	// 4. Switch to External Mode
	// CreateDefaultWasmFileClientIfNotExist should switch mode and compile to disk
	// (Note: source already exists, so it skips generation but should switch)
	c.CreateDefaultWasmFileClientIfNotExist()

	if c.strategy.Name() != "External" {
		t.Errorf("Expected External strategy after switch, got %s", c.strategy.Name())
	}

	// Verify file WAS written to disk
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Error("Expected file on disk after switching to External mode")
	}

	// 5. Verify serving in External Mode
	// Note: We need to re-register routes if the strategy instance changed?
	// WasmClient.RegisterRoutes delegates to w.strategy.RegisterRoutes.
	// But previously registered handlers on 'mux' are bound to the OLD strategy instance (closure).
	// Ideally, the app re-registers or the handler delegates dynamically.
	// In our implementation, `RegisterRoutes` calls `mux.HandleFunc`.
	// For this test, we create a new mux to verify the NEW strategy.
	mux2 := http.NewServeMux()
	c.RegisterRoutes(mux2)
	ts2 := httptest.NewServer(mux2)
	defer ts2.Close()

	url2 := ts2.URL + "/assets/test-client.wasm"
	resp2, err := http.Get(url2)
	if err != nil {
		t.Fatalf("Failed to GET %s: %v", url2, err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp2.StatusCode)
	}
}
