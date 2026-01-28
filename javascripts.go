package client

import (
	_ "embed"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	. "github.com/tinywasm/fmt"
)

//go:embed assets/wasm_exec_go.js
var embeddedWasmExecGo []byte

//go:embed assets/wasm_exec_tinygo.js
var embeddedWasmExecTinyGo []byte

// wasm_execGoSignatures returns signatures expected in Go's wasm_exec.js
func wasm_execGoSignatures() []string {
	return []string{
		"runtime.scheduleTimeoutEvent",
		"runtime.clearTimeoutEvent",
		"runtime.wasmExit",
		// note: removed shared or ambiguous signatures such as syscall/js.valueGet
	}
}

// wasm_execTinyGoSignatures returns signatures expected in TinyGo's wasm_exec.js
func wasm_execTinyGoSignatures() []string {
	return []string{
		"runtime.sleepTicks",
		"runtime.ticks",
		"$runtime.alloc",
		"tinygo_js",
	}
}

// WasmExecJsOutputPath returns the output path for wasm_exec.js
func (w *WasmClient) WasmExecJsOutputPath() string {
	return path.Join(w.appRootDir, w.wasmExecJsOutputDir, "wasm_exec.js")
}

// getWasmExecContent returns the raw wasm_exec.js content for the current compiler configuration.
// This method returns the unmodified content from embedded assets without any headers or caching.
// It relies on WasmClient's internal state (via WasmProjectTinyGoJsUse) to determine which
// compiler (Go vs TinyGo) to use.
//
// The returned content is suitable for:
//   - Direct file output
//   - Integration into build tools
//   - Embedding in worker scripts
//
// Note: This method does NOT add mode headers or perform caching. Those responsibilities
// belong to GetSSRClientInitJS() which is used for the internal initialization flow.
func (w *WasmClient) getWasmExecContent(mode string) ([]byte, error) {
	// Determine project type and compiler from WasmClient state
	isWasm, useTinyGo := w.WasmProjectTinyGoJsUse(mode)
	if !isWasm {
		return nil, Errf("not a WASM project")
	}

	// DEBUG: Log which wasm_exec.js is being selected
	//w.Logger("DEBUG getWasmExecContent: mode=", mode, ", useTinyGo=", useTinyGo, ", buildMediumShortcut=", w.buildMediumSizeShortcut)

	// Return appropriate embedded content based on compiler configuration
	if useTinyGo {
		//w.Logger("DEBUG: Returning TinyGo wasm_exec.js")
		return embeddedWasmExecTinyGo, nil
	}
	//w.Logger("DEBUG: Returning Go standard wasm_exec.js")
	return embeddedWasmExecGo, nil
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
//   - GetSSRClientInitJS("// Custom Header\n", "console.log('loaded');") - Both custom
func (h *WasmClient) GetSSRClientInitJS(customizations ...string) (js string, err error) {
	mode := h.Value()
	isWasm, _ := h.WasmProjectTinyGoJsUse(mode)
	if !isWasm {
		return "", nil // Not a WASM project
	}

	// Always regenerate the JS, do not use cache

	// Get raw content from embedded assets instead of system paths
	wasmJs, err := h.getWasmExecContent(mode)
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

	// Verify activeSizeBuilder is initialized before accessing it
	if h.activeSizeBuilder == nil {
		return "", Errf("activeSizeBuilder not initialized")
	}

	// Determine footer: use custom if provided, otherwise default
	var footer string
	if len(customizations) > 1 {
		footer = customizations[1]
	} else {
		// Default footer: WebAssembly initialization code
		footer = `
		const go = new Go();
		WebAssembly.instantiateStreaming(fetch("` + h.activeSizeBuilder.MainOutputFileNameWithExtension() + `"), go.importObject).then((result) => {
			go.run(result.instance);
		});
	`
	}
	stringWasmJs += footer

	// Normalize JS output to avoid accidental differences between cached and
	// freshly-generated content (line endings, trailing spaces).
	normalized := normalizeJs(stringWasmJs)

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
		if h.tinyGoCompiler {
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

// GetWasmExecJsPathTinyGo returns the path to TinyGo's wasm_exec.js file
func (w *WasmClient) GetWasmExecJsPathTinyGo() (string, error) {
	// Method 1: Try standard lib location pattern
	libPaths := []string{
		"/usr/local/lib/tinygo/targets/wasm_exec.js",
		"/opt/tinygo/targets/wasm_exec.js",
	}

	for _, path := range libPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Method 2: Derive from tinygo executable path
	tinygoPath, err := exec.LookPath("tinygo")
	if err != nil {
		return "", Errf("tinygo executable not found: %v", err)
	}

	// Get directory where tinygo is located
	tinyGoDir := filepath.Dir(tinygoPath)

	// Common installation patterns
	patterns := []string{
		// For /usr/local/bin/tinygo -> /usr/local/lib/tinygo/targets/wasm_exec.js
		filepath.Join(filepath.Dir(tinyGoDir), "lib", "tinygo", "targets", "wasm_exec.js"),
		// For /usr/bin/tinygo -> /usr/lib/tinygo/targets/wasm_exec.js
		filepath.Join(filepath.Dir(tinyGoDir), "lib", "tinygo", "targets", "wasm_exec.js"),
		// For portable installation: remove bin and add targets
		filepath.Join(filepath.Dir(tinyGoDir), "targets", "wasm_exec.js"),
	}

	for _, wasmExecPath := range patterns {
		if _, err := os.Stat(wasmExecPath); err == nil {
			return wasmExecPath, nil
		}
	}

	return "", Errf("TinyGo wasm_exec.js not found. Searched paths: %v", append(libPaths, patterns...))
}

// GetWasmExecJsPathGo returns the path to Go's wasm_exec.js file
func (w *WasmClient) GetWasmExecJsPathGo() (string, error) {
	// Method 1: Try GOROOT environment variable (most reliable)
	goRoot := os.Getenv("GOROOT")
	if goRoot != "" {
		patterns := []string{
			filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js"), // Traditional location
			filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"),  // Modern location
		}
		for _, wasmExecPath := range patterns {
			if _, err := os.Stat(wasmExecPath); err == nil {
				return wasmExecPath, nil
			}
		}
	}

	// Method 2: Derive from go executable path
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", Errf("go executable not found: %v", err)
	}

	// Get installation directory (parent of bin directory)
	goDir := filepath.Dir(goPath)

	// Remove bin directory from path (cross-platform)
	if filepath.Base(goDir) == "bin" {
		goDir = filepath.Dir(goDir)
	}

	// Try multiple patterns for different Go versions
	patterns := []string{
		filepath.Join(goDir, "misc", "wasm", "wasm_exec.js"), // Traditional location
		filepath.Join(goDir, "lib", "wasm", "wasm_exec.js"),  // Modern location (Go 1.21+)
	}

	for _, wasmExecPath := range patterns {
		if _, err := os.Stat(wasmExecPath); err == nil {
			return wasmExecPath, nil
		}
	}

	return "", Errf("go wasm_exec.js not found. Searched: GOROOT=%s, patterns=%v", goRoot, patterns)
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
