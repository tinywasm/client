package client

import (
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
	if w.Database != nil {
		w.Database.Set(StoreKeySizeMode, newValue)
	}

	// Auto-recompile
	if err := w.RecompileMainWasm(); err != nil {
		errorMsg := Translate("Error:", "auto", "compilation", "failed:", err).String()
		if errorMsg == "" {
			errorMsg = "Error: auto compilation failed: " + err.Error()
		}
		progress <- errorMsg
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

// RecompileMainWasm recompiles the main WASM file using the current storage mode.
func (w *WasmClient) RecompileMainWasm() error {
	if w.storage == nil {
		return Err("storage not initialized")
	}

	// Use storage.Compile() to respect In-Memory vs Disk mode
	return w.storage.Compile()
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
