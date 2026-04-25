package client

import (
	_ "embed"
	"flag"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	. "github.com/tinywasm/fmt"
)

//go:embed assets/wasm_exec_go.js
var embeddedWasmExecGo []byte

//go:embed assets/wasm_exec_tinygo.js
var embeddedWasmExecTinyGo []byte

func init() {
	flag.String("wasmsize_mode", "", "wasm size mode (passed by tinywasm)")
}

// WasmExecGoSignatures returns signatures expected in Go's wasm_exec.js
func WasmExecGoSignatures() []string {
	return []string{
		"runtime.scheduleTimeoutEvent",
		"runtime.clearTimeoutEvent",
		"runtime.wasmExit",
		// note: removed shared or ambiguous signatures such as syscall/js.valueGet
	}
}

// WasmExecTinyGoSignatures returns signatures expected in TinyGo's wasm_exec.js
func WasmExecTinyGoSignatures() []string {
	return []string{
		"runtime.sleepTicks",
		"runtime.ticks",
		"$runtime.alloc",
		"tinygo_js",
	}
}

// Javascript provides functionalities to generate WASM initialization JavaScript.
// It can be used independently or embedded in other structures.
// Javascript provides functionalities to generate WASM initialization JavaScript.
// It can be used independently or embedded in other structures.
type Javascript struct {
	useTinyGo    bool
	wasmFilename string
	wasmSizeMode string
}

// SetMode sets the compilation mode and automatically determines if TinyGo is needed.
func (j *Javascript) SetMode(mode string) {
	j.wasmSizeMode = mode
	// Logic: "M" and "S" modes imply TinyGo. "L" implies Go (standard).
	j.useTinyGo = (mode == "M" || mode == "S")
}

// SetWasmFilename sets the WASM filename to be used in the generated JavaScript.
func (j *Javascript) SetWasmFilename(filename string) {
	j.wasmFilename = filename
}

// NewJavascriptFromArgs creates a new Javascript instance by parsing command line arguments.
func NewJavascriptFromArgs() *Javascript {
	j := &Javascript{
		wasmFilename: "client.wasm",
	}
	mode := ParseWasmSizeModeFlag()
	j.SetMode(mode)
	return j
}

// RegisterRoutes registers the WASM file route on the provided mux.
// The route path is derived from WasmFilename (e.g., "/client.wasm").
func (j *Javascript) RegisterRoutes(mux *http.ServeMux, wasmFilePath string) {
	wasmFile := j.wasmFilename
	if wasmFile == "" {
		wasmFile = "client.wasm"
	}

	routePath := "/" + wasmFile

	mux.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, wasmFilePath)
	})
}

// ArgumentsForServer returns runtime arguments for the server,
// relying solely on the -wasmsize_mode flag.
func (j *Javascript) ArgumentsForServer() []string {
	return []string{
		Sprintf("-wasmsize_mode=%s", j.wasmSizeMode),
	}
}

// ParseWasmSizeModeFlag parses -wasmsize_mode flag from os.Args.
// Returns the value found, or empty string if not present.
func ParseWasmSizeModeFlag() string {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-wasmsize_mode=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// WasmExecJsOutputPath returns the output path for wasm_exec.js
func (w *WasmClient) WasmExecJsOutputPath() string {
	return path.Join(w.AppRootDir, w.wasmExecJsOutputDir, "wasm_exec.js")
}

// getWasmExecContent returns the raw wasm_exec.js content.
func (j *Javascript) getWasmExecContent() ([]byte, error) {
	if j.useTinyGo {
		return embeddedWasmExecTinyGo, nil
	}
	return embeddedWasmExecGo, nil
}

// getWasmExecContent returns the raw wasm_exec.js content for the current compiler configuration.
// This method returns the unmodified content from embedded assets without any headers or caching.
// It relies on WasmClient's internal state (via WasmProjectTinyGoJsUse) to determine which
// compiler (Go vs TinyGo) to use.
func (w *WasmClient) getWasmExecContent(mode string) ([]byte, error) {
	// Determine project type and compiler from WasmClient state
	isWasm, _ := w.WasmProjectTinyGoJsUse(mode)
	if !isWasm {
		return nil, Errf("not a WASM project")
	}

	w.Javascript.SetMode(mode) // Update mode and useTinyGo internal state
	return w.Javascript.getWasmExecContent()
}

// GetSSRClientInitJS returns the JavaScript code needed to initialize WASM.
func (j *Javascript) GetSSRClientInitJS(customizations ...string) (js string, err error) {
	wasmJs, err := j.getWasmExecContent()
	if err != nil {
		return "", err
	}

	stringWasmJs := string(wasmJs)

	// Determine header: use custom if provided, otherwise default
	var header string
	if len(customizations) > 0 {
		header = customizations[0]
	}

	stringWasmJs = header + stringWasmJs

	// Determine footer: use custom if provided, otherwise default
	var footer string
	if len(customizations) > 1 {
		footer = customizations[1]
	} else {
		// Default footer: WebAssembly initialization code
		wasmFile := j.wasmFilename
		if wasmFile == "" {
			wasmFile = "client.wasm"
		}
		footer = `
		const go = new Go();
		WebAssembly.instantiateStreaming(fetch("` + wasmFile + `"), go.importObject).then((result) => {
			go.run(result.instance);
		});
	`
	}
	stringWasmJs += footer

	// Normalize JS output to avoid accidental differences between cached and
	// freshly-generated content (line endings, trailing spaces).
	return normalizeJs(stringWasmJs), nil
}

// GetSSRClientInitJS returns the JavaScript code needed to initialize WASM.
//
// Parameters (variadic):
//   - customizations[0]: Custom header string to prepend to wasm_exec.js content.
//     If not provided, defaults to "// WasmClient: mode=<current_mode>\n"
//   - customizations[1]: Custom footer string to append after wasm_exec.js content.
//     If not provided, defaults to WebAssembly initialization code with fetch and instantiate.
//
// Examples:
//   - GetSSRClientInitJS() - Uses default header and footer
//   - GetSSRClientInitJS("// Custom Header\n") - Custom header, default footer
//   - GetSSRClientInitJS("// Custom Header\n", "console.Log('loaded');") - Both custom
func (h *WasmClient) GetSSRClientInitJS(customizations ...string) (js string, err error) {
	mode := h.Value()
	isWasm, _ := h.WasmProjectTinyGoJsUse(mode)
	if !isWasm {
		return "", nil // Not a WASM project
	}

	// Always regenerate the JS, do not use cache

	// Verify activeSizeBuilder is initialized before accessing it
	if h.activeSizeBuilder == nil {
		return "", Errf("activeSizeBuilder not initialized")
	}

	h.Javascript.SetMode(mode)
	h.Javascript.SetWasmFilename(h.activeSizeBuilder.MainOutputFileNameWithExtension())

	normalized, err := h.Javascript.GetSSRClientInitJS(customizations...)
	if err != nil {
		return "", err
	}

	// Store in appropriate cache based on mode
	switch mode {
	case h.buildLargeSizeShortcut:
		h.mode_large_go_wasm_exec_cache = normalized
	case h.buildMediumSizeShortcut:
		h.mode_medium_tinygo_wasm_exec_cache = normalized
	case h.buildSmallSizeShortcut:
		h.mode_small_tinygo_wasm_exec_cache = normalized
	default:
		// Fallback: if TinyGo compiler in use write to tinyGo cache, otherwise go cache
		if h.TinyGoCompilerFlag {
			h.mode_medium_tinygo_wasm_exec_cache = normalized
		} else {
			h.mode_large_go_wasm_exec_cache = normalized
		}
	}

	return normalized, nil
}

// normalizeJs applies deterministic normalization to JS content so cached
// and regenerated outputs are identical: convert CRLF to LF and trim trailing
// whitespace from each line.
func normalizeJs(s string) string {
	// Normalize CRLF -> LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Trim trailing whitespace on each line
	lines := strings.Split(s, "\n")
	for i, L := range lines {
		lines[i] = strings.TrimRight(L, " \t")
	}
	return strings.Join(lines, "\n")
}

// ClearJavaScriptCache clears both cached JavaScript strings to force regeneration
func (h *WasmClient) ClearJavaScriptCache() {
	h.mode_large_go_wasm_exec_cache = ""
	h.mode_medium_tinygo_wasm_exec_cache = ""
	h.mode_small_tinygo_wasm_exec_cache = ""
}



// wasmProjectWriteOrReplaceWasmExecJsOutput writes (or overwrites) the
// wasm_exec.js initialization file into the configured web output folder.
// On success or on any write attempt it returns true; any
// filesystem or generation errors are logged via w.Logger and treated as
// non-fatal so callers can continue their workflow.
func (w *WasmClient) wasmProjectWriteOrReplaceWasmExecJsOutput() {
	outputPath := w.WasmExecJsOutputPath()

	//w.Logger("DEBUG: Writing/overwriting wasm_exec.js to output path:", outputPath)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		w.Logger("Failed to create output directory:", err)
		return // We did attempt the operation (project), but treat errors as non-fatal
	}

	// Get the complete JavaScript initialization code (includes WASM setup)
	jsContent, err := w.GetSSRClientInitJS()
	if err != nil {
		w.Logger("Failed to generate JavaScript initialization code:", err)
		return
	}

	// Write the complete JavaScript to output location, always overwrite
	if err := os.WriteFile(outputPath, []byte(jsContent), 0644); err != nil {
		w.Logger("Failed to write JavaScript initialization file:", err)
		return
	}

	//w.Logger("DEBUG: Wrote/overwrote JavaScript initialization file in output directory")
}
