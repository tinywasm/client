package client

import (
	. "github.com/tinywasm/fmt"
)

func (w *WasmClient) Shortcuts() []map[string]string {
	return []map[string]string{
		{w.buildLargeSizeShortcut: Translate("mode", "Large", "stLib").String()},
		{w.buildMediumSizeShortcut: Translate("mode", "Medium", "tinygo").String()},
		{w.buildSmallSizeShortcut: Translate("mode", "Small", "tinygo").String()},
	}
}

// Change updates the compiler mode for WasmClient.
// Implements the HandlerEdit interface: Change(newValue string)
func (w *WasmClient) Change(newValue string) {
	// Normalize input: trim spaces and convert to uppercase
	newValue = Convert(newValue).ToUpper().String()

	// Validate mode
	if err := w.ValidateMode(newValue); err != nil {
		w.Logger(err.Error())
		return
	}

	// Lazily verify TinyGo installation status ONLY when a TinyGo mode is requested
	if w.RequiresTinyGo(newValue) {
		w.verifyTinyGoInstallationStatus()
		if !w.TinyGoInstalled {
			w.Logger(w.handleTinyGoMissing().Error())
			return
		}
	}

	// Update active builder
	w.UpdateCurrentBuilder(newValue)

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
	if w.EnableWasmExecJsOutput {
		w.wasmProjectWriteOrReplaceWasmExecJsOutput()
	}

	// Only notify listener when compilation succeeded.
	// If compilation failed, the new mode's wasm_exec.js would mismatch with the
	// old mode's .wasm binary, causing the browser to freeze on reload.
	if compilationSuccess && w.OnWasmExecChange != nil {
		w.OnWasmExecChange()
	}

	if compilationSuccess {
		w.LogSuccessState("Changed", "To", "Mode", newValue)
	}
}

// RecompileMainWasm recompiles the main WASM file using the current Storage mode.
func (w *WasmClient) RecompileMainWasm() error {
	if w.Storage == nil {
		return Err("Storage not initialized")
	}

	// Use Storage.Compile() to respect In-Memory vs Disk mode
	return w.Storage.Compile()
}

// ValidateMode validates if the provided mode is supported
func (w *WasmClient) ValidateMode(mode string) error {
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

	return Err("mode", ":", mode, "invalid", "valid", ":", validModes)
}

// LogSuccessState logs the standard success message with WASM details (Safe: Acquires Lock)
func (w *WasmClient) LogSuccessState(messages ...any) {

	args := append(messages, "WASM", w.Storage.Name(), w.MainInputFileRelativePath(), w.activeSizeBuilder.BinarySize())
	w.Logger(args...)
}
