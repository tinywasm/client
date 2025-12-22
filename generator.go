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
func (t *WasmClient) CreateDefaultWasmFileClientIfNotExist() *WasmClient {
	// Build target path from Config
	targetPath := filepath.Join(t.appRootDir, t.Config.SourceDir, t.mainInputFile)

	// Never overwrite existing files
	if _, err := os.Stat(targetPath); err == nil {
		if t.Logger != nil {
			t.Logger("WASM source file already exists at", targetPath, ", skipping generation")
		}
		// Fallthrough to switch mode logic below
	} else {
		// Read embedded markdown (no template processing needed - static content)
		raw, errRead := embeddedFS.ReadFile("templates/basic_wasm_client.md")
		if errRead != nil {
			if t.Logger != nil {
				t.Logger("Error reading embedded template:", errRead)
			}
			return t
		}

		// Use devflow to extract Go code from markdown
		writer := func(name string, data []byte) error {
			if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				return err
			}
			return os.WriteFile(name, data, 0o644)
		}

		// devflow needs the full destination path
		destDir := filepath.Join(t.appRootDir, t.Config.SourceDir)

		m := devflow.NewMarkDown(t.appRootDir, destDir, writer).
			InputByte(raw)

		if t.Logger != nil {
			m.SetLogger(t.Logger)
		}

		// Extract to the main file
		if err := m.Extract(t.mainInputFile); err != nil {
			if t.Logger != nil {
				t.Logger("Error extracting go code from markdown:", err)
			}
			return t
		}

		if t.Logger != nil {
			t.Logger("Generated WASM source file at", targetPath)
		}

		t.wasmProject = true

		t.VisualStudioCodeWasmEnvConfig()
	}

	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	// Skip if DisableWasmExecJsOutput is set (e.g., for inline embedding scenarios)
	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	// Skip if DisableWasmExecJsOutput is set (e.g., for inline embedding scenarios)
	// Ensure wasm_exec.js is present in output (create/overwrite as needed)
	if t.enableWasmExecJsOutput {
		t.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	// Switch to External Mode (Persistent)
	// This ensures subsequent compilations write to disk
	t.strategy = &externalStrategy{client: t}
	//t.Logger("Switched to External Mode (Disk)")

	// Trigger initial compilation to disk
	if err := t.strategy.Compile(); err != nil {
		t.Logger("Initial compilation failed:", err)
	}

	return t
}
