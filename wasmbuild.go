package client

import (
	"os"
	"path/filepath"

	. "github.com/tinywasm/fmt"
)

// WasmBuildArgs defines the arguments for the RunWasmBuild function.
type WasmBuildArgs struct {
	Stdlib bool // true = Go standard compiler mode "L", false = TinyGo mode "S"
}

// RunWasmBuild performs the common logic for the wasmbuild CLI.
func RunWasmBuild(args WasmBuildArgs) error {
	// 1. If not stdlib: call EnsureTinyGoInstalled() and add to PATH
	if !args.Stdlib {
		tinyGoPath, err := EnsureTinyGoInstalled()
		if err != nil {
			return Errf("error ensuring TinyGo installation: %w", err)
		}

		if tinyGoPath != "" {
			// Add to PATH so exec.LookPath (used by gobuild or os/exec) can find it
			newPath := filepath.Dir(tinyGoPath) + string(os.PathListSeparator) + os.Getenv("PATH")
			os.Setenv("PATH", newPath)
		}
	}

	// 2. Verify input: check that web/client.go exists
	inputPath := filepath.Join("web", "client.go")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Errf("input file not found: %s", inputPath)
	}

	// 3. Create output dir: web/public
	outputDir := filepath.Join("web", "public")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return Errf("failed to create output directory: %w", err)
	}

	// 4. Generate script.js
	mode := "S"
	if args.Stdlib {
		mode = "L"
	}

	js := Javascript{}
	js.SetMode(mode)
	js.SetWasmFilename("client.wasm")
	jsContent, err := js.GetSSRClientInitJS()
	if err != nil {
		return Errf("failed to generate script.js: %w", err)
	}

	scriptPath := filepath.Join(outputDir, "script.js")
	if err := os.WriteFile(scriptPath, []byte(jsContent), 0644); err != nil {
		return Errf("failed to write script.js: %w", err)
	}

	// 5. Compile WASM
	cfg := NewConfig()
	// NewConfig() defaults should be SourceDir="web" and OutputDir="web/public",
	// but we explicitly set them based on the required layout for safety.
	cfg.SourceDir = func() string { return "web" }
	cfg.OutputDir = func() string { return outputDir }

	w := New(cfg)
	w.SetMode(mode)
	w.SetBuildOnDisk(true, false)
	w.SetLog(Println)

	if err := w.Compile(); err != nil {
		return Errf("WASM compilation failed: %w", err)
	}

	w.LogSuccessState("compiled")

	return nil
}
