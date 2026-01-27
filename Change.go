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
	if err := w.RecompileMainWasm(); err != nil {
		errorMsg := Translate("Error:", "auto", "compilation", "failed:", err).String()
		if errorMsg == "" {
			errorMsg = "Error: auto compilation failed: " + err.Error()
		}
		w.Logger(errorMsg)
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
	switch newValue {
	case w.buildLargeSizeShortcut:
		w.LogSuccessState("Changed", "To", "Mode", "Large")
	case w.buildMediumSizeShortcut:
		w.LogSuccessState("Changed", "To", "Mode", "Medium")
	case w.buildSmallSizeShortcut:
		w.LogSuccessState("Changed", "To", "Mode", "Small")
	default:
		w.LogSuccessState("Changed", "To", "Mode", newValue)
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

// LogSuccessState logs the standard success message with WASM details
func (w *WasmClient) LogSuccessState(messages ...any) {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	w.logSuccessState(messages...)
}

func (w *WasmClient) logSuccessState(messages ...any) {
	s := w.storage
	if s == nil {
		return
	}

	args := append(messages, "WASM", s.Name(), w.MainInputFileRelativePath(), w.activeSizeBuilder.BinarySize())
	w.Logger(args...)
}
