package client

import (
	"os"
	"path/filepath"
	"sync"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/tinygo"
)

// RunWasmBuildClient captures the subset of WasmClient methods used by RunWasmBuild.
// It is exported so tests can provide lightweight fakes without pulling in gobuild.
type RunWasmBuildClient interface {
	SetMode(string)
	SetBuildOnDisk(bool, bool)
	SetLog(func(...any))
	Compile() error
	LogSuccessState(...any)
}

type runWasmBuildDeps struct {
	ensureTinyGoInstalled func() (string, error)
	tinyGoEnv             func() []string
	newClient             func(*Config) RunWasmBuildClient
}

var (
	wasmBuildDeps = runWasmBuildDeps{
		ensureTinyGoInstalled: func() (string, error) {
			return tinygo.EnsureInstalled()
		},
		tinyGoEnv: func() []string {
			return tinygo.GetEnv()
		},
		newClient: func(cfg *Config) RunWasmBuildClient {
			return New(cfg)
		},
	}
	wasmBuildDepsMu sync.Mutex
)

// RunWasmBuildHooks lets tests override RunWasmBuild dependencies (installer, env provider, client factory).
// Use SetRunWasmBuildHooks in tests to temporarily replace these functions.
type RunWasmBuildHooks struct {
	EnsureTinyGoInstalled func() (string, error)
	TinyGoEnv             func() []string
	NewClient             func(*Config) RunWasmBuildClient
}

// SetRunWasmBuildHooks updates RunWasmBuild dependencies for the duration of a test.
// It returns a restore function that should be deferred.
func SetRunWasmBuildHooks(h RunWasmBuildHooks) (restore func()) {
	wasmBuildDepsMu.Lock()
	prev := wasmBuildDeps
	if h.EnsureTinyGoInstalled != nil {
		wasmBuildDeps.ensureTinyGoInstalled = h.EnsureTinyGoInstalled
	}
	if h.TinyGoEnv != nil {
		wasmBuildDeps.tinyGoEnv = h.TinyGoEnv
	}
	if h.NewClient != nil {
		wasmBuildDeps.newClient = h.NewClient
	}
	wasmBuildDepsMu.Unlock()

	return func() {
		wasmBuildDepsMu.Lock()
		wasmBuildDeps = prev
		wasmBuildDepsMu.Unlock()
	}
}

// WasmBuildArgs defines the arguments for the RunWasmBuild function.
type WasmBuildArgs struct {
	Stdlib bool // true = Go standard compiler mode "L", false = TinyGo mode "S"
}

// RunWasmBuild performs the common logic for the wasmbuild CLI.
func RunWasmBuild(args WasmBuildArgs) error {
	// 1. If not stdlib: call EnsureTinyGoInstalled() and get env from tinygo package
	if !args.Stdlib {
		_, err := wasmBuildDeps.ensureTinyGoInstalled()
		if err != nil {
			return Errf("error ensuring TinyGo installation: %w", err)
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
	// Get environment with TINYGOROOT and updated PATH (safe for subprocess injection)
	cfg := NewConfig()
	if !args.Stdlib {
		cfg.Env = wasmBuildDeps.tinyGoEnv()
	}
	// NewConfig() defaults should be SourceDir="web" and OutputDir="web/public",
	// but we explicitly set them based on the required layout for safety.
	cfg.SourceDir = func() string { return "web" }
	cfg.OutputDir = func() string { return outputDir }

	w := wasmBuildDeps.newClient(cfg)
	w.SetMode(mode)
	w.SetBuildOnDisk(true, false)
	w.SetLog(Println)

	if err := w.Compile(); err != nil {
		return Errf("WASM compilation failed: %w", err)
	}

	w.LogSuccessState("compiled")

	return nil
}
