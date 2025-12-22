package client

import (
	"net/http"
	"os"
	"path/filepath"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/gobuild"
)

// StoreKeySizeMode is the key used to store the current compiler mode in the Store
const StoreKeySizeMode = "wasmsize_mode"

// WasmClient provides WebAssembly compilation capabilities with 3-mode compiler selection
type WasmClient struct {
	*Config

	// RENAME & ADD: 4 builders for complete mode coverage
	builderSizeLarge  *gobuild.GoBuild // Go standard - fast compilation
	builderSizeMedium *gobuild.GoBuild // TinyGo debug - easier debugging
	builderSizeSmall  *gobuild.GoBuild // TinyGo production - smallest size
	activeSizeBuilder *gobuild.GoBuild // Current active builder

	// EXISTING: Keep for installation detection (no compilerMode needed - activeSizeBuilder handles state)
	tinyGoCompiler  bool // Enable TinyGo compiler (default: false for faster development)
	wasmProject     bool // Automatically detected based on file structure
	tinyGoInstalled bool // Cached TinyGo installation status

	// NEW: Explicit mode tracking to fix Value() method
	currenSizeMode string // Track current mode explicitly ("L", "M", "S")

	mode_large_go_wasm_exec_cache      string // cache wasm_exec.js file content per mode large
	mode_medium_tinygo_wasm_exec_cache string // cache wasm_exec.js file content per mode medium
	mode_small_tinygo_wasm_exec_cache  string // cache wasm_exec.js file content per mode small

	storage BuildStorage // Storage for compilation and serving (In-Memory vs External)

	wasmExecJsOutputDir string // output dir for wasm_exec.js file (relative) eg: "web/js", "theme/js"

	// Configuration fields moved from Config
	appRootDir              string
	mainInputFile           string
	outputName              string
	buildLargeSizeShortcut  string
	buildMediumSizeShortcut string
	buildSmallSizeShortcut  string
	enableWasmExecJsOutput  bool // Default: false (disabled)
	lastOpID                string
}

// New creates a new WasmClient instance with the provided configuration
// Timeout is set to 40 seconds maximum as TinyGo compilation can be slow
// Default values: MainInputFile in Config defaults to "main.wasm.go"
func New(c *Config) *WasmClient {
	// Ensure we have a config
	defaults := NewConfig()
	if c == nil {
		c = defaults
	}

	if c.Logger == nil {
		c.Logger = defaults.Logger
	}

	w := &WasmClient{
		Config: c,

		// Initialize dynamic fields
		tinyGoCompiler:  false, // Default to fast Go compilation; enable later via WasmClient methods if desired
		wasmProject:     false, // Auto-detected later
		tinyGoInstalled: false, // Verified on first use

		// Initialize with proper defaults (not from Config anymore)
		appRootDir:              ".",
		mainInputFile:           "client.go",
		outputName:              "client",
		buildLargeSizeShortcut:  "L",
		buildMediumSizeShortcut: "M",
		buildSmallSizeShortcut:  "S",
		enableWasmExecJsOutput:  false,

		// Initialize with default mode
		currenSizeMode: "L", // Start with coding mode
	}

	// Initialize gobuild instance with WASM-specific configuration
	w.builderWasmInit()

	// Try to restore mode from store if available
	w.loadMode()

	// Default to In-Memory storage
	w.storage = &memoryStorage{client: w}

	// Perform one-time detection at the end
	w.detectProjectConfiguration()

	return w
}

// RegisterRoutes registers the WASM client file route on the provided mux.
// It delegates to the active storage.
func (w *WasmClient) RegisterRoutes(mux *http.ServeMux) {
	w.storage.RegisterRoutes(mux)
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
		return "/" + prefix + "/" + w.outputName + ".wasm"
	}
	return "/" + w.outputName + ".wasm"
}

// Name returns the name of the WASM project
func (w *WasmClient) Name() string {
	return "CLIENT"
}

// WasmProjectTinyGoJsUse returns dynamic state based on current configuration
func (w *WasmClient) WasmProjectTinyGoJsUse(mode ...string) (isWasmProject bool, useTinyGo bool) {
	var currenSizeMode string
	if len(mode) > 0 {
		currenSizeMode = mode[0]
	} else {
		currenSizeMode = w.Value()
	}

	useTinyGo = w.requiresTinyGo(currenSizeMode)

	return w.wasmProject, useTinyGo
}

// === DevTUI FieldHandler Interface Implementation ===

// Label returns the field label for DevTUI display
func (w *WasmClient) Label() string {
	return "Compiler Mode"
}

// Value returns the current compiler mode shortcut (c, d, or p)
func (w *WasmClient) Value() string {
	// Sync with store if available
	w.loadMode()

	// Use explicit mode tracking instead of pointer comparison
	if w.currenSizeMode == "" {
		return w.buildLargeSizeShortcut // Default to coding mode
	}
	return w.currenSizeMode
}

// SetBuildOnDisk switches between In-Memory and External (Disk) storage.
func (w *WasmClient) SetBuildOnDisk(onDisk bool) {
	if onDisk {
		if _, ok := w.storage.(*diskStorage); !ok {
			w.storage = &diskStorage{client: w}
			w.Logger("WASM Client switched to External (Disk) Mode")
		}
	} else {
		if _, ok := w.storage.(*memoryStorage); !ok {
			w.storage = &memoryStorage{client: w}
			w.Logger("WASM Client switched to In-Memory Mode")
		}
	}
	// Trigger immediate compilation to ensure the new storage has fresh content
	if err := w.storage.Compile(); err != nil {
		w.Logger("Compilation failed after mode switch:", err)
	}
}

// loadMode updates currenSizeMode from the store if available
func (w *WasmClient) loadMode() {
	if w.Store != nil {
		if val, err := w.Store.Get(StoreKeySizeMode); err == nil && val != "" {
			w.currenSizeMode = val
		}
	}
}

// SetWasmExecJsOutputDir sets the output directory for wasm_exec.js.
// This is primarily intended for tests/debug where physical file output is required.
// Setting a non-empty path will trigger a project detection and, if detected,
// write/update the wasm_exec.js file to that directory.
func (w *WasmClient) SetWasmExecJsOutputDir(path string) {
	w.wasmExecJsOutputDir = path
	w.detectProjectConfiguration()
	if w.wasmProject && w.enableWasmExecJsOutput && path != "" {
		w.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}
}

// SetAppRootDir sets the application root directory (absolute).
func (w *WasmClient) SetAppRootDir(path string) {
	w.appRootDir = path
	w.builderWasmInit()
	w.detectProjectConfiguration()
}

// SetMainInputFile sets the main input file for WASM compilation (default: "client.go").
func (w *WasmClient) SetMainInputFile(file string) {
	w.mainInputFile = file
	w.builderWasmInit()
	w.detectProjectConfiguration()
}

// SetOutputName sets the output name for WASM file (default: "client").
func (w *WasmClient) SetOutputName(name string) {
	w.outputName = name
	w.builderWasmInit()
	w.detectProjectConfiguration()
}

// SetBuildShortcuts sets the shortcuts for the three compilation modes.
// If an empty string is provided for a shortcut, it remains unchanged.
func (w *WasmClient) SetBuildShortcuts(large, medium, small string) {
	if large != "" {
		w.buildLargeSizeShortcut = large
	}
	if medium != "" {
		w.buildMediumSizeShortcut = medium
	}
	if small != "" {
		w.buildSmallSizeShortcut = small
	}

	// Update current mode if it was one of the shortcuts and we changed it?
	// actually currenSizeMode is just a string, it might need update if we changed the shortcut it currently uses.
	// But usually this is called once during init.
}

// SetEnableWasmExecJsOutput enables automatic creation of wasm_exec.js file.
func (w *WasmClient) SetEnableWasmExecJsOutput(enable bool) {
	w.enableWasmExecJsOutput = enable
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
		return
	}

	w.Logger("No WASM project detected")
}

// detectFromGoFiles checks for .wasm.go files to confirm WASM project
func (w *WasmClient) detectFromGoFiles() bool {
	// Walk the project directory to find .wasm.go files
	wasmFilesFound := false

	err := filepath.Walk(w.appRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if info.IsDir() {
			return nil // Continue walking directories
		}

		// Get relative path from AppRootDir for comparison
		relPath, err := filepath.Rel(w.appRootDir, path)
		if err != nil {
			relPath = path // Fallback to absolute path if relative fails
		}

		fileName := info.Name()

		// Check for main input file in the source directory (strong indicator of WASM project)
		expectedPath := filepath.Join(w.Config.SourceDir, w.mainInputFile)
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
