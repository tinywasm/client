package client_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/client"
	"github.com/tinywasm/router/mock"
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

	cfg := &client.Config{
		SourceDir:       func() string { return "web" },
		OutputDir:       func() string { return "web/public" },
		AssetsURLPrefix: "assets",
	}

	// 1. Initialize - Should be In-Memory (no .wasm file yet)
	c := client.New(cfg)
	c.SetLog(func(msg ...any) {
		t.Log(msg...)
	})
	c.SetAppRootDir(tmpDir)
	c.SetMainInputFile("client.go")
	c.SetOutputName("test-client")

	if c.Storage.Name() != "In-Memory" {
		t.Errorf("Expected In-Memory client.Storage, got %s", c.Storage.Name())
	}

	// 2. Trigger Event -> Compile to Memory
	// This should populate the internal buffer
	err = c.NewFileEvent("client.go", ".go", srcFile, "write")
	if err != nil {
		t.Fatalf("NewFileEvent failed: %v", err)
	}

	// 3. Register Routes and Verify with mock router
	r := &mock.Router{}
	c.RegisterRoutes(r)

	// Verify the route was registered
	routes := r.Routes()
	if len(routes) == 0 {
		t.Fatal("Expected at least one route registered")
	}
	if routes[0].Path != "/assets/test-client.wasm" {
		t.Errorf("Expected path /assets/test-client.wasm, got %s", routes[0].Path)
	}

	// Verify MemoryStorage is still active and has content
	if c.Storage.Name() != "In-Memory" {
		t.Errorf("Expected In-Memory storage, got %s", c.Storage.Name())
	}

	// Verify file was NOT written to disk
	diskPath := filepath.Join(tmpDir, "web/public/test-client.wasm")
	if _, err := os.Stat(diskPath); err == nil {
		t.Error("Expected no file on disk in In-Memory mode")
	}

	// 4. Switch to External Mode
	c.CreateDefaultWasmFileClientIfNotExist(false)
	c.UseDiskStorage()
	if err := c.Compile(); err != nil {
		t.Fatalf("Compile failed after switching to DiskStorage: %v", err)
	}

	if c.Storage.Name() != "External" {
		t.Errorf("Expected External client.Storage after switch, got %s", c.Storage.Name())
	}

	// Verify file WAS written to disk
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Error("Expected file on disk after switching to External mode")
	}

	// 5. Verify External Mode registration
	r2 := &mock.Router{}
	c.RegisterRoutes(r2)

	// Verify the route is still registered correctly
	routes2 := r2.Routes()
	if len(routes2) == 0 {
		t.Fatal("Expected route registered in External mode")
	}
	if routes2[0].Path != "/assets/test-client.wasm" {
		t.Errorf("Expected path /assets/test-client.wasm in External mode, got %s", routes2[0].Path)
	}
}
