package client

// KeyValueDataBase defines the interface for a key-value storage system
// used to persist the compiler state (e.g. current mode).
type KeyValueDataBase interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

// Config holds configuration for WASM compilation
type Config struct {
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

	Logger func(message ...any)
	// TinyGoCompiler removed: tinyGoCompiler (private) in WasmClient is used instead to avoid confusion

	// gobuild integration fields
	Callback           func(error)     // Optional callback for async compilation
	CompilingArguments func() []string // Build arguments for compilation (e.g., ldflags)

	Database         KeyValueDataBase // Key-Value store for state persistence
	OnWasmExecChange func()           // Callback for wasm_exec.js changes
}

// NewConfig creates a WasmClient Config with sensible defaults
func NewConfig() *Config {
	return &Config{
		SourceDir: "web",
		OutputDir: "web/public",
		Logger: func(message ...any) {
			// Default logger: do nothing (silent operation)
		},
	}
}
