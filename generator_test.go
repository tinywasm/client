package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDefaultWasmFileClientIfNotExistCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	sourceDir := "src/cmd/webclient"
	fullSourcePath := filepath.Join(tmp, sourceDir)

	cfg := NewConfig()
	cfg.SourceDir = func() string { return sourceDir }

	tw := New(cfg)
	tw.SetAppRootDir(tmp)
	tw.SetMainInputFile("main.go")
	tw.SetShouldGenerateDefaultFile(func() bool { return true })

	// Ensure no existing file
	target := filepath.Join(fullSourcePath, "main.go")
	if _, err := os.Stat(target); err == nil {
		t.Fatalf("expected no existing file at %s", target)
	}

	result := tw.CreateDefaultWasmFileClientIfNotExist()
	if result == nil {
		t.Fatalf("CreateDefaultWasmFileClientIfNotExist returned nil")
	}

	// Verify file was created
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	contentStr := string(content)

	// Verify basic content
	if !strings.Contains(contentStr, "package main") {
		t.Errorf("generated file missing package main")
	}
	if !strings.Contains(contentStr, "syscall/js") {
		t.Errorf("generated file missing syscall/js import")
	}
	if !strings.Contains(contentStr, "Hello from WebAssembly!") {
		t.Errorf("generated file missing expected message")
	}
	if !strings.Contains(contentStr, `createElement`) {
		t.Errorf("generated file missing createElement call")
	}
	if !strings.Contains(contentStr, `select {}`) {
		t.Errorf("generated file missing select statement")
	}
}

func TestCreateDefaultWasmFileClientIfNotExistDoesNotOverwrite(t *testing.T) {
	tmp := t.TempDir()
	sourceDir := "src/cmd/webclient"
	fullSourcePath := filepath.Join(tmp, sourceDir)
	if err := os.MkdirAll(fullSourcePath, 0755); err != nil {
		t.Fatalf("creating source dir: %v", err)
	}

	cfg := NewConfig()
	cfg.SourceDir = func() string { return sourceDir }

	tw := New(cfg)
	tw.SetAppRootDir(tmp)
	tw.SetMainInputFile("main.go")
	tw.SetShouldGenerateDefaultFile(func() bool { return true })

	target := filepath.Join(fullSourcePath, "main.go")

	// Create existing file with different content
	original := "// ORIGINAL CONTENT DO NOT OVERWRITE"
	if err := os.WriteFile(target, []byte(original), 0644); err != nil {
		t.Fatalf("writing original file: %v", err)
	}

	// Try to generate (should skip)
	result := tw.CreateDefaultWasmFileClientIfNotExist()
	if result == nil {
		t.Fatalf("CreateDefaultWasmFileClientIfNotExist returned nil")
	}

	// Verify original content is preserved
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading file after generate: %v", err)
	}

	if string(content) != original {
		t.Fatalf("file was overwritten, expected original content")
	}
}
