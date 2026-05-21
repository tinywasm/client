package client

import (
	"sync"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/gobuild"
	"github.com/tinywasm/tinygo"
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
	TinyGoCompilerFlag bool // Enable TinyGo compiler (default: false for faster development)
	TinyGoInstalled bool // Cached TinyGo installation status

	// NEW: Explicit mode tracking to fix Value() method
	CurrentSizeMode string // Track current mode explicitly ("L", "M", "S")

	Storage BuildStorage // Storage for compilation and serving (In-Memory vs External)

	// Configuration fields moved from Config
	AppRootDir string
	MainInputFile string
	OutputName string
	buildLargeSizeShortcut    string
	buildMediumSizeShortcut   string
	buildSmallSizeShortcut    string
	ShouldCreateIDEConfig func() bool
	ShouldGenerateDefaultFile func() bool
	Log func(message ...any)

	// OnCompile is invoked after each compilation triggered by NewFileEvent.
	// err==nil indicates success; err!=nil indicates failure.
	OnCompile func(err error)

	// storageMu protects Storage and CurrentSizeMode fields from concurrent access
	storageMu sync.RWMutex
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
		TinyGoCompilerFlag:  false, // Default to fast Go compilation; enable later via WasmClient methods if desired
		TinyGoInstalled: false, // Verified on first use

		// Initialize with proper defaults (not from Config anymore)
		AppRootDir:              ".",
		MainInputFile:           "client.go",
		OutputName:              "client",
		buildLargeSizeShortcut:  "L",
		buildMediumSizeShortcut: "M",
		buildSmallSizeShortcut:  "S",

		// Initialize with default mode
		CurrentSizeMode: "L", // Start with coding mode

		ShouldCreateIDEConfig:     func() bool { return false },
		ShouldGenerateDefaultFile: func() bool { return false },
	}

	// Initialize gobuild instance with WASM-specific configuration
	w.builderWasmInit()

	// Try to restore mode from store if available
	w.loadMode()

	// Default to In-Memory Storage
	w.Storage = &MemoryStorage{Client: w}

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
		return "/" + prefix + "/" + w.OutputName + ".wasm"
	}
	return "/" + w.OutputName + ".wasm"
}

// Name returns the name of the WASM project
func (w *WasmClient) Name() string {
	return "CLIENT"
}

func (w *WasmClient) SetLog(f func(message ...any)) {
	w.Log = f
}

func (w *WasmClient) Logger(messages ...any) {
	if w.Log != nil {
		w.Log(messages...)
	}
}

// WasmProjectTinyGoJsUse returns dynamic state based on current configuration
func (w *WasmClient) WasmProjectTinyGoJsUse(mode ...string) (isWasmProject bool, useTinyGo bool) {
	var CurrentSizeMode string
	if len(mode) > 0 {
		CurrentSizeMode = mode[0]
	} else {
		CurrentSizeMode = w.Value()
	}

	useTinyGo = w.RequiresTinyGo(CurrentSizeMode)

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
	// w.loadMode() // REMOVED: Causes race condition/reversion if DB is stale. Mode is managed in memory via Change().

	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	// Use explicit mode tracking instead of pointer comparison
	if w.CurrentSizeMode == "" {
		return w.buildLargeSizeShortcut // Default to coding mode
	}
	return w.CurrentSizeMode
}

// UseDiskStorage switches the client to disk-backed storage. Idempotent.
// Does NOT trigger compilation — the caller composes Compile() when needed.
func (w *WasmClient) UseDiskStorage() {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()

	if _, ok := w.Storage.(*DiskStorage); ok {
		return
	}

	w.Storage = &DiskStorage{Client: w}
	w.LogSuccessState("Changed", "To", "Storage", "External")
}

// UseMemoryStorage switches the client to in-memory storage. Idempotent.
// Provided for symmetry and test usage; production code does not call this
// (memory is the default at construction).
func (w *WasmClient) UseMemoryStorage() {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()

	if _, ok := w.Storage.(*MemoryStorage); ok {
		return
	}

	w.Storage = &MemoryStorage{Client: w}
	w.LogSuccessState("Changed", "To", "Storage", "In-Memory")
}

// loadMode updates CurrentSizeMode from the store if available and syncs the active builder
func (w *WasmClient) loadMode() {
	if w.Database != nil {
		if val, err := w.Database.Get(StoreKeySizeMode); err == nil && val != "" {
			w.storageMu.Lock()
			defer w.storageMu.Unlock()
			// Only update if the mode is different from current
			if w.CurrentSizeMode != val {
				w.CurrentSizeMode = val
				// Sync the active builder with the loaded mode
				// This ensures the correct compiler (Go vs TinyGo) is used
				w.UpdateCurrentBuilder(val)
			}
		}
	}
}

// SetAppRootDir sets the application root directory (absolute).
func (w *WasmClient) SetAppRootDir(path string) {
	w.AppRootDir = path
	w.builderWasmInit()
}

// SetMainInputFile sets the main input file for WASM compilation (default: "client.go").
func (w *WasmClient) SetMainInputFile(file string) {
	w.MainInputFile = file
	w.builderWasmInit()
}

// SetOutputName sets the output name for WASM file (default: "client").
func (w *WasmClient) SetOutputName(name string) {
	w.OutputName = name
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
	// actually CurrentSizeMode is just a string, it might need update if we changed the shortcut it currently uses.
	// But usually this is called once during init.
}

// SetShouldCreateIDEConfig sets a function that determines if IDE configuration
// files (like .vscode) should be created.
func (w *WasmClient) SetShouldCreateIDEConfig(f func() bool) {
	w.ShouldCreateIDEConfig = f
}

// SetShouldGenerateDefaultFile sets a function that determines if the default
// WASM client source file (usually client.go) should be created if it doesn't exist.
func (w *WasmClient) SetShouldGenerateDefaultFile(f func() bool) {
	w.ShouldGenerateDefaultFile = f
}

// UseTinyGo returns true if the current mode requires TinyGo's wasm_exec.js
func (w *WasmClient) UseTinyGo() bool {
	_, useTinyGo := w.WasmProjectTinyGoJsUse()
	return useTinyGo
}

// ArgumentsForServer returns runtime args to pass to the server,
// including the -wasmsize_mode flag based on current compiler mode.
func (w *WasmClient) ArgumentsForServer() []string {
	return []string{
		Sprintf("-wasmsize_mode=%s", w.Value()),
	}
}

// VerifyTinyGoInstallation checks if TinyGo is properly installed
func (w *WasmClient) VerifyTinyGoInstallation() error {
	if tinygo.IsInstalled() {
		return nil
	}
	return Err("TinyGo", "not", "found")
}

// GetTinyGoVersion returns the installed TinyGo version
func (w *WasmClient) GetTinyGoVersion() (string, error) {
	return tinygo.GetVersion()
}
