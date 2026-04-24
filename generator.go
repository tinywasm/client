package client

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/tinywasm/devflow"
)

//go:embed templates/*
var embeddedFS embed.FS

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

		m := devflow.NewMarkDown(t.AppRootDir, srcDir, writer).
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

		// Trigger compilation immediately so In-Memory mode has content to serve
		t.storageMu.RLock()
		store := t.Storage
		t.storageMu.RUnlock()

		if store != nil {
			if err := store.Compile(); err != nil {
				t.Logger("Error compiling generated client:", err)
			}
		}
	}

	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	// Skip if DisableWasmExecJsOutput is set (e.g., for inline embedding scenarios)
	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	// Skip if DisableWasmExecJsOutput is set (e.g., for inline embedding scenarios)
	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	if t.EnableWasmExecJsOutput {
		t.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	if t.EnableWasmExecJsOutput {
		t.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	return t
}
