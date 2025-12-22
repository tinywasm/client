package client

import (
	"os"
	"path"

	. "github.com/tinywasm/fmt"
)

func (w *WasmClient) Shortcuts() []map[string]string {
	return []map[string]string{
		{w.buildLargeSizeShortcut: Translate(D.Mode, "Large", "stLib").String()},
		{w.buildMediumSizeShortcut: Translate(D.Mode, "Medium", "tinygo").String()},
		{w.buildSmallSizeShortcut: Translate(D.Mode, "Small", "tinygo").String()},
	}
}

// Change updates the compiler mode for WasmClient and reports progress via the provided channel.
// Implements the HandlerEdit interface: Change(newValue string, progress chan<- string)
// NOTE: The caller (devtui) is responsible for closing the progress channel, NOT the handler.
func (w *WasmClient) Change(newValue string, progress chan<- string) {
	// DO NOT close the channel - devtui owns it and will close it after this method returns
	// Normalize input: trim spaces and convert to uppercase
	newValue = Convert(newValue).ToUpper().String()

	// Validate mode
	if err := w.validateMode(newValue); err != nil {
		progress <- err.Error()
		return
	}

	// Lazily verify TinyGo installation status ONLY when a TinyGo mode is requested
	if w.requiresTinyGo(newValue) {
		w.verifyTinyGoInstallationStatus()
		if !w.tinyGoInstalled {
			progress <- w.handleTinyGoMissing().Error()
			return
		}
	}

	// Update active builder
	w.updateCurrentBuilder(newValue)

	// Save mode to store if available
	if w.Store != nil {
		w.Store.Set("tinywasm_mode", newValue)
	}

	// Check if main WASM file exists
	sourceDir := path.Join(w.appRootDir, w.Config.SourceDir)
	mainWasmPath := path.Join(sourceDir, w.mainInputFile)
	if _, err := os.Stat(mainWasmPath); err != nil {
		progress <- w.getSuccessMessage(newValue) // Changed from progress(...)
		return
	}

	// Auto-recompile
	if err := w.RecompileMainWasm(); err != nil {
		warningMsg := Translate("Warning:", "auto", "compilation", "failed:", err).String()
		if warningMsg == "" {
			warningMsg = "Warning: auto compilation failed: " + err.Error()
		}
		progress <- warningMsg // Changed from progress(warningMsg)
		return
	}

	// Ensure wasm_exec.js is available
	if w.enableWasmExecJsOutput {
		w.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	// Notify listener about change
	if w.OnWasmExecChange != nil {
		w.OnWasmExecChange()
	}

	// Report success
	progress <- w.getSuccessMessage(newValue)
}

// RecompileMainWasm recompiles the main WASM file if it exists
func (w *WasmClient) RecompileMainWasm() error {
	if w.activeBuilder == nil {
		return Err("builder not initialized")
	}
	sourceDir := path.Join(w.appRootDir, w.Config.SourceDir)
	mainWasmPath := path.Join(sourceDir, w.mainInputFile)

	// Check if main.wasm.go exists
	if _, err := os.Stat(mainWasmPath); err != nil {
		return Err("main WASM file not found:", mainWasmPath)
	}

	// Use gobuild to compile
	return w.activeBuilder.CompileProgram()
}

// validateMode validates if the provided mode is supported
func (w *WasmClient) validateMode(mode string) error {
	// Ensure mode is uppercase to match configured shortcuts which are
	// expected to be single uppercase letters by default.
	mode = Convert(mode).ToUpper().String()

	validModes := []string{
		Convert(w.buildLargeSizeShortcut).ToUpper().String(),
		Convert(w.buildMediumSizeShortcut).ToUpper().String(),
		Convert(w.buildSmallSizeShortcut).ToUpper().String(),
	}

	for _, valid := range validModes {
		if mode == valid {
			return nil
		}
	}

	return Err(D.Mode, ":", mode, D.Invalid, D.Valid, ":", validModes)
}

// getSuccessMessage returns appropriate success message for mode
func (w *WasmClient) getSuccessMessage(mode string) string {

	switch mode {
	case w.buildLargeSizeShortcut:
		return Translate(D.Changed, D.To, D.Mode, "Large").String()
	case w.buildMediumSizeShortcut:
		return Translate(D.Changed, D.To, D.Mode, "Medium").String()
	case w.buildSmallSizeShortcut:
		return Translate(D.Changed, D.To, D.Mode, "Small").String()
	default:
		return Translate(D.Mode, ":", mode, D.Invalid).String()
	}

}

func (w *WasmClient) GetLastOperationID() string { return w.lastOpID }
