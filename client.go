package client

import (
	"net/http"
	"os"
	"path/filepath"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/gobuild"
)

// WasmClient provides WebAssembly compilation capabilities with 3-mode compiler selection
type WasmClient struct {
	*Config

	// RENAME & ADD: 4 builders for complete mode coverage
	builderLarge  *gobuild.GoBuild // Go standard - fast compilation
	builderMedium *gobuild.GoBuild // TinyGo debug - easier debugging
	builderSmall  *gobuild.GoBuild // TinyGo production - smallest size
	activeBuilder *gobuild.GoBuild // Current active builder

	// EXISTING: Keep for installation detection (no compilerMode needed - activeBuilder handles state)
	tinyGoCompiler  bool // Enable TinyGo compiler (default: false for faster development)
	wasmProject     bool // Automatically detected based on file structure
	tinyGoInstalled bool // Cached TinyGo installation status

	// NEW: Explicit mode tracking to fix Value() method
	currentMode string // Track current mode explicitly ("L", "M", "S")

	mode_large_go_wasm_exec_cache      string // cache wasm_exec.js file content per mode large
	mode_medium_tinygo_wasm_exec_cache string // cache wasm_exec.js file content per mode medium
	mode_small_tinygo_wasm_exec_cache  string // cache wasm_exec.js file content per mode small

	strategy ClientStrategy // Strategy for compilation and serving (In-Memory vs External)
}

// Config holds configuration for WASM compilation
type Config struct {

	// AppRootDir specifies the application root directory (absolute).
	// e.g., "/home/user/project". If empty, defaults to "." to preserve existing behavior.
	AppRootDir string

	// SourceDir specifies the directory containing the Go source for the webclient (relative to AppRootDir).
	// e.g., "web"
	SourceDir string

	// OutputDir specifies the directory for WASM and related assets (relative to AppRootDir).
	// e.g., "web/public"
	OutputDir string

	// AssetsURLPrefix is an optional URL prefix/folder for serving the WASM file.
	// e.g. "assets" -> serves at "/assets/client.wasm"
	// default: "" -> serves at "/client.wasm"
	AssetsURLPrefix string

	WasmExecJsOutputDir string // output dir for wasm_exec.js file (relative) eg: "web/js", "theme/js"
	MainInputFile       string // main input file for WASM compilation (default: "client.go")
	OutputName          string // output name for WASM file (default: "client")
	Logger              func(message ...any)
	// TinyGoCompiler removed: tinyGoCompiler (private) in WasmClient is used instead to avoid confusion

	BuildLargeSizeShortcut  string // "L" (Large) compile with go
	BuildMediumSizeShortcut string // "M" (Medium) compile with tinygo debug
	BuildSmallSizeShortcut  string // "S" (Small) compile with tinygo minimal binary size

	// gobuild integration fields
	Callback           func(error)     // Optional callback for async compilation
	CompilingArguments func() []string // Build arguments for compilation (e.g., ldflags)

	// DisableWasmExecJsOutput prevents automatic creation of wasm_exec.js file
	// Useful when embedding wasm_exec.js content inline (e.g., Cloudflare Pages Advanced Mode)
	DisableWasmExecJsOutput bool

	// LastOperationID tracks the last operation ID for progress reporting
	lastOpID string

	Store            Store  // Key-Value store for state persistence
	OnWasmExecChange func() // Callback for wasm_exec.js changes
}

// NewConfig creates a WasmClient Config with sensible defaults
func NewConfig() *Config {
	return &Config{
		AppRootDir:              ".",
		SourceDir:               "web",
		OutputDir:               "web/public",
		WasmExecJsOutputDir:     "web/js",
		MainInputFile:           "client.go",
		OutputName:              "client",
		BuildLargeSizeShortcut:  "L",
		BuildMediumSizeShortcut: "M",
		BuildSmallSizeShortcut:  "S",
		Logger: func(message ...any) {
			// Default logger: do nothing (silent operation)
		},
	}
}

// New creates a new WasmClient instance with the provided configuration
// Timeout is set to 40 seconds maximum as TinyGo compilation can be slow
// Default values: MainInputFile in Config defaults to "main.wasm.go"
func New(c *Config) *WasmClient {
	// Ensure we have a config and a default AppRootDir
	if c == nil {
		c = NewConfig()
	}
	if c.AppRootDir == "" {
		c.AppRootDir = "."
	}

	// Set default logger if not provided
	if c.Logger == nil {
		c.Logger = func(message ...any) {
			// Default logger: do nothing (silent operation)
		}
	}

	// Ensure shortcut defaults are set even when a partial config is passed
	// Use NewConfig() as the authoritative source of defaults and copy any
	// missing shortcut values from it.
	defaults := NewConfig()
	if c.BuildLargeSizeShortcut == "" {
		c.BuildLargeSizeShortcut = defaults.BuildLargeSizeShortcut
	}
	if c.BuildMediumSizeShortcut == "" {
		c.BuildMediumSizeShortcut = defaults.BuildMediumSizeShortcut
	}
	if c.BuildSmallSizeShortcut == "" {
		c.BuildSmallSizeShortcut = defaults.BuildSmallSizeShortcut
	}
	if c.MainInputFile == "" {
		c.MainInputFile = defaults.MainInputFile
	}
	if c.OutputName == "" {
		c.OutputName = defaults.OutputName
	}

	w := &WasmClient{
		Config: c,

		// Initialize dynamic fields
		tinyGoCompiler:  false, // Default to fast Go compilation; enable later via WasmClient methods if desired
		wasmProject:     false, // Auto-detected later
		tinyGoInstalled: false, // Verified on first use

		// Initialize with default mode
		currentMode: c.BuildLargeSizeShortcut, // Start with coding mode
	}

	if w.currentMode == "" {
		w.currentMode = w.Config.BuildLargeSizeShortcut
	}

	// Set default for WasmExecJsOutputDir if not configured
	if w.Config.WasmExecJsOutputDir == "" {
		w.Config.WasmExecJsOutputDir = "src/web/ui/js"
	}

	// Initialize gobuild instance with WASM-specific configuration
	w.builderWasmInit()

	// Try to restore mode from store if available
	if w.Store != nil {
		if val, err := w.Store.Get("tinywasm_mode"); err == nil && val != "" {
			w.currentMode = val
		}
	}

	// Determine initial strategy
	// If the external WASM file already exists, use External strategy.
	// Otherwise, default to In-Memory strategy.
	outputFile := filepath.Join(w.Config.OutputDir, w.Config.OutputName+".wasm")
	absOutputFile := filepath.Join(w.Config.AppRootDir, outputFile)

	if _, err := os.Stat(absOutputFile); err == nil {
		w.strategy = &externalStrategy{client: w}
		//w.Logger("WASM Client initialized in External Mode (file found)")
	} else {
		w.strategy = &inMemoryStrategy{client: w}
		//w.Logger("WASM Client initialized in In-Memory Mode (default)")
	}

	// Perform one-time detection at the end
	w.detectProjectConfiguration()

	return w
}

// RegisterRoutes registers the WASM client file route on the provided mux.
// It delegates to the active strategy.
func (w *WasmClient) RegisterRoutes(mux *http.ServeMux) {
	w.strategy.RegisterRoutes(mux)
}

// wasmRoutePath calculates the URL path for the WASM file
func (w *WasmClient) wasmRoutePath() string {
	prefix := w.Config.AssetsURLPrefix
	// Ensure safe joining of URL paths
	if prefix != "" {
		// Clean the prefix
		if prefix[0] == '/' {
			prefix = prefix[1:]
		}
		if prefix[len(prefix)-1] == '/' {
			prefix = prefix[:len(prefix)-1]
		}
		return "/" + prefix + "/" + w.Config.OutputName + ".wasm"
	}
	return "/" + w.Config.OutputName + ".wasm"
}

// Name returns the name of the WASM project
func (w *WasmClient) Name() string {
	return "WasmClient"
}

// WasmProjectTinyGoJsUse returns dynamic state based on current configuration
func (w *WasmClient) WasmProjectTinyGoJsUse(mode ...string) (isWasmProject bool, useTinyGo bool) {
	var currentMode string
	if len(mode) > 0 {
		currentMode = mode[0]
	} else {
		currentMode = w.Value()
	}

	useTinyGo = w.requiresTinyGo(currentMode)

	return w.wasmProject, useTinyGo
}

// === DevTUI FieldHandler Interface Implementation ===

// Label returns the field label for DevTUI display
func (w *WasmClient) Label() string {
	return "Compiler Mode"
}

// Value returns the current compiler mode shortcut (c, d, or p)
func (w *WasmClient) Value() string {
	// Use explicit mode tracking instead of pointer comparison
	if w.currentMode == "" {
		return w.Config.BuildLargeSizeShortcut // Default to coding mode
	}
	return w.currentMode
}

// detectProjectConfiguration performs one-time detection during initialization
func (w *WasmClient) detectProjectConfiguration() {
	// Priority 1: Check for existing wasm_exec.js (definitive source)
	if w.detectFromExistingWasmExecJs() {
		//w.Logger("DEBUG: WASM project detected from existing wasm_exec.js")
		return
	}

	// Priority 2: Check for .go files (confirms WASM project)
	if w.detectFromGoFiles() {
		w.wasmProject = true
		// If a project is detected from .go files, it means there's no wasm_exec.js,
		// so we should create it.
		if !w.Config.DisableWasmExecJsOutput {
			w.wasmProjectWriteOrReplaceWasmExecJsOutput()
		}
		return
	}

	w.Logger("No WASM project detected")
}

// detectFromGoFiles checks for .wasm.go files to confirm WASM project
func (w *WasmClient) detectFromGoFiles() bool {
	// Walk the project directory to find .wasm.go files
	wasmFilesFound := false

	err := filepath.Walk(w.Config.AppRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if info.IsDir() {
			return nil // Continue walking directories
		}

		// Get relative path from AppRootDir for comparison
		relPath, err := filepath.Rel(w.Config.AppRootDir, path)
		if err != nil {
			relPath = path // Fallback to absolute path if relative fails
		}

		fileName := info.Name()

		// Check for main input file in the source directory (strong indicator of WASM project)
		expectedPath := filepath.Join(w.Config.SourceDir, w.Config.MainInputFile)
		if relPath == expectedPath {
			wasmFilesFound = true
			return filepath.SkipAll // Found main file, can stop walking
		}

		// Check for .wasm.go files in modules (another strong indicator)
		if HasSuffix(fileName, ".wasm.go") {
			wasmFilesFound = true
			return filepath.SkipAll // Found wasm file, can stop walking
		}

		return nil
	})

	if err != nil {
		w.Logger("Error walking directory for WASM file detection:", err)
		return false
	}

	return wasmFilesFound
}
