package client

import (
	"sync"

	"github.com/tinywasm/gobuild"
)

// StoreKeySizeMode is the key used to store the current compiler mode in the Database
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
	// EXISTING: Keep for installation detection (no compilerMode needed - activeSizeBuilder handles state)
	tinyGoCompiler  bool // Enable TinyGo compiler (default: false for faster development)
	tinyGoInstalled bool // Cached TinyGo installation status

	// NEW: Explicit mode tracking to fix Value() method
	currenSizeMode string // Track current mode explicitly ("L", "M", "S")

	mode_large_go_wasm_exec_cache      string // cache wasm_exec.js file content per mode large
	mode_medium_tinygo_wasm_exec_cache string // cache wasm_exec.js file content per mode medium
	mode_small_tinygo_wasm_exec_cache  string // cache wasm_exec.js file content per mode small

	storage BuildStorage // Storage for compilation and serving (In-Memory vs External)

	wasmExecJsOutputDir string // output dir for wasm_exec.js file (relative) eg: "web/js", "theme/js"

	// Configuration fields moved from Config
	appRootDir                string
	mainInputFile             string
	outputName                string
	buildLargeSizeShortcut    string
	buildMediumSizeShortcut   string
	buildSmallSizeShortcut    string
	enableWasmExecJsOutput    bool // Default: false (disabled)
	shouldCreateIDEConfig     func() bool
	shouldGenerateDefaultFile func() bool
	log                       func(message ...any)

	// storageMu protects storage and currenSizeMode fields from concurrent access
	storageMu sync.RWMutex

	// Javascript provides WASM initialization JS snippets
	Javascript *Javascript
}

// New creates a new WasmClient instance with the provided configuration
func New(c *Config) *WasmClient {
	if c == nil {
		c = NewConfig()
	}

	// Ensure dynamic fields are never nil to prevent panics in builders
	if c.SourceDir == nil {
		c.SourceDir = func() string { return "web" }
	}
	if c.OutputDir == nil {
		c.OutputDir = func() string { return "web/public" }
	}

	w := &WasmClient{
		Config: c,

		// Initialize dynamic fields
		tinyGoCompiler:  false, // Default to fast Go compilation; enable later via WasmClient methods if desired
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

		shouldCreateIDEConfig:     func() bool { return false },
		shouldGenerateDefaultFile: func() bool { return false },
	}

	// Initialize gobuild instance with WASM-specific configuration
	w.builderWasmInit()

	// Try to restore mode from store if available
	w.loadMode()

	// Default to In-Memory storage
	w.storage = &memoryStorage{client: w}

	w.Javascript = &Javascript{
		WasmFilename: w.outputName + ".wasm",
	}

	return w
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

func (w *WasmClient) SetLog(f func(message ...any)) {
	w.log = f
}

func (w *WasmClient) Logger(messages ...any) {
	if w.log != nil {
		w.log(messages...)
	}
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

	return true, useTinyGo
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

	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	// Use explicit mode tracking instead of pointer comparison
	if w.currenSizeMode == "" {
		return w.buildLargeSizeShortcut // Default to coding mode
	}
	return w.currenSizeMode
}

// SetBuildOnDisk switches between In-Memory and External (Disk) storage.
// When compileNow is true, compilation is triggered immediately after mode switch.
// When compileNow is false, compilation (and directory creation) is deferred until first explicit Compile() call.
func (w *WasmClient) SetBuildOnDisk(onDisk, compileNow bool) {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()

	var newModeName string
	var newStorage BuildStorage

	if onDisk {
		if _, ok := w.storage.(*diskStorage); ok {
			return
		}
		newModeName = "External"
		newStorage = &diskStorage{client: w}
	} else {
		if _, ok := w.storage.(*memoryStorage); ok {
			return
		}
		newModeName = "In-Memory"
		newStorage = &memoryStorage{client: w}
	}

	// Apply switch
	w.storage = newStorage

	if compileNow {
		if err := w.storage.Compile(); err != nil {
			w.Logger("Compilation failed after mode switch:", err)
			return
		}
	}

	w.logSuccessState("Changed", "To", "Storage", newModeName)
}

// loadMode updates currenSizeMode from the store if available and syncs the active builder
func (w *WasmClient) loadMode() {
	if w.Database != nil {
		if val, err := w.Database.Get(StoreKeySizeMode); err == nil && val != "" {
			w.storageMu.Lock()
			defer w.storageMu.Unlock()
			// Only update if the mode is different from current
			if w.currenSizeMode != val {
				w.currenSizeMode = val
				// Sync the active builder with the loaded mode
				// This ensures the correct compiler (Go vs TinyGo) is used
				w.updateCurrentBuilder(val)
			}
		}
	}
}

// SetWasmExecJsOutputDir sets the output directory for wasm_exec.js.
// This is primarily intended for tests/debug where physical file output is required.
// Setting a non-empty path will trigger a write/update of the wasm_exec.js file to that directory.
func (w *WasmClient) SetWasmExecJsOutputDir(path string) {
	w.wasmExecJsOutputDir = path
	if w.enableWasmExecJsOutput && path != "" {
		w.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}
}

// SetAppRootDir sets the application root directory (absolute).
func (w *WasmClient) SetAppRootDir(path string) {
	w.appRootDir = path
	w.builderWasmInit()
}

// SetMainInputFile sets the main input file for WASM compilation (default: "client.go").
func (w *WasmClient) SetMainInputFile(file string) {
	w.mainInputFile = file
	w.builderWasmInit()
}

// SetOutputName sets the output name for WASM file (default: "client").
func (w *WasmClient) SetOutputName(name string) {
	w.outputName = name
	w.builderWasmInit()
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

// SetShouldCreateIDEConfig sets a function that determines if IDE configuration
// files (like .vscode) should be created.
func (w *WasmClient) SetShouldCreateIDEConfig(f func() bool) {
	w.shouldCreateIDEConfig = f
}

// SetShouldGenerateDefaultFile sets a function that determines if the default
// WASM client source file (usually client.go) should be created if it doesn't exist.
func (w *WasmClient) SetShouldGenerateDefaultFile(f func() bool) {
	w.shouldGenerateDefaultFile = f
}

// UseTinyGo returns true if the current mode requires TinyGo's wasm_exec.js
func (w *WasmClient) UseTinyGo() bool {
	_, useTinyGo := w.WasmProjectTinyGoJsUse()
	return useTinyGo
}

// ArgumentsForServer returns runtime args to pass to the server,
// including the -usetinygo flag based on current compiler mode.
func (w *WasmClient) ArgumentsForServer() []string {
	w.Javascript.UseTinyGo = w.UseTinyGo()
	return w.Javascript.ArgumentsForServer()
}
