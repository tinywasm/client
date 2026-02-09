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

// Change updates the compiler mode for WasmClient.
// Implements the HandlerEdit interface: Change(newValue string)
func (w *WasmClient) Change(newValue string) {
	// Normalize input: trim spaces and convert to uppercase
	newValue = Convert(newValue).ToUpper().String()

	// Validate mode
	if err := w.validateMode(newValue); err != nil {
		w.Logger(err.Error())
		return
	}

	// Lazily verify TinyGo installation status ONLY when a TinyGo mode is requested
	if w.requiresTinyGo(newValue) {
		w.verifyTinyGoInstallationStatus()
		if !w.tinyGoInstalled {
			w.Logger(w.handleTinyGoMissing().Error())
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
	compilationSuccess := true
	if err := w.RecompileMainWasm(); err != nil {
		errorMsg := Translate("Error:", "auto", "compilation", "failed:", err).String()
		//errorMsg = "Error: auto compilation failed: " + err.Error()
		w.Logger(errorMsg)
		compilationSuccess = false
		// Don't return early - still need to update assets and notify listeners
	}

	// Ensure wasm_exec.js is available
	if w.enableWasmExecJsOutput {
		w.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	// IMPORTANT: Always notify listener about mode change, even if compilation failed
	// The assets (wasm_exec.js) need to be regenerated to match the new mode
	if w.OnWasmExecChange != nil {
		w.OnWasmExecChange()
	}

	if compilationSuccess {
		w.logSuccessState("Changed", "To", "Mode", newValue)
	}
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

// logSuccessState logs the standard success message with WASM details (Safe: Acquires Lock)
func (w *WasmClient) logSuccessState(messages ...any) {

	args := append(messages, "WASM", w.storage.Name(), w.MainInputFileRelativePath(), w.activeSizeBuilder.BinarySize())
	w.Logger(args...)
}
