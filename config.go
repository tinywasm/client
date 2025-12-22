package client

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

	MainInputFile string // main input file for WASM compilation (default: "client.go")
	OutputName    string // output name for WASM file (default: "client")
	Logger        func(message ...any)
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
