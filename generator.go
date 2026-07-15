package client

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/tinywasm/command"
	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/markdown"
)

//go:embed templates/*
var embeddedFS embed.FS

// templateModules: modules imported by templates/basic_wasm_client.md.
// Keep in sync with the template's imports.
var templateModules = []string{
	"github.com/tinywasm/dom",
	"github.com/tinywasm/fmt",
	"github.com/tinywasm/html",
}

// CreateDefaultWasmFileClientIfNotExist creates a default WASM main.go file from the embedded markdown template
// It never overwrites an existing file and returns the WasmClient instance for method chaining.
func (t *WasmClient) CreateDefaultWasmFileClientIfNotExist(skipIDEConfig bool) *WasmClient {
	// Check if generation is allowed
	if t.ShouldGenerateDefaultFile != nil && !t.ShouldGenerateDefaultFile() {
		return t
	}

	// Path to client.go
	clientPath := filepath.Join(t.AppRootDir, t.Config.SourceDir(), t.MainInputFile)
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		// Read embedded markdown (no template processing needed - static content)
		raw, errRead := embeddedFS.ReadFile("templates/basic_wasm_client.md")
		if errRead != nil {
			t.Logger("Error reading embedded template:", errRead)
			return t
		}

		// Use devflow to extract Go code from markdown
		writer := func(name string, data []byte) error {
			if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				return err
			}
			return os.WriteFile(name, data, 0o644)
		}

		// Ensure SourceDir exists
		srcDir := filepath.Join(t.AppRootDir, t.Config.SourceDir())
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			if err := os.MkdirAll(srcDir, 0o755); err != nil {
				t.Logger("Error creating source directory:", err)
				return t
			}
		}

		m := markdown.New(t.AppRootDir, srcDir, writer).
			InputByte(raw)

		// Extract to the main file
		if err := m.Extract(t.MainInputFile); err != nil {
			t.Logger("Error extracting go code from markdown:", err)
			return t
		}

		t.LogSuccessState("Generated WASM source file at", clientPath)

		if !skipIDEConfig {
			t.VisualStudioCodeWasmEnvConfig()
		}

		// Ensure dependencies are present before compiling
		if err := t.ensureTemplateDependencies(); err != nil {
			t.Logger("Error ensuring template dependencies:", err)
			t.storageMu.Lock()
			t.lastBuildError = err
			t.storageMu.Unlock()
			return t
		}

		// Trigger compilation immediately so In-Memory mode has content to serve
		t.storageMu.RLock()
		store := t.Storage
		t.storageMu.RUnlock()

		if store != nil {
			if err := store.Compile(); err != nil {
				t.Logger("Error compiling generated client:", err)
				t.storageMu.Lock()
				t.lastBuildError = err
				t.storageMu.Unlock()
			} else {
				t.storageMu.Lock()
				t.lastBuildError = nil
				t.storageMu.Unlock()
			}
		}
	}

	return t
}

func (t *WasmClient) ensureTemplateDependencies() error {
	for _, mod := range templateModules {
		t.Logger("Ensuring dependency:", mod)
		_, err := command.RunInDir(t.AppRootDir, "go", "get", mod+"@latest")
		if err != nil {
			return Errf("failed to add dependency %s: %w. Please run: go get %s@latest", mod, err, mod)
		}
	}
	return nil
}
