package client

import (
	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/fmt/lang"
)

func (w *WasmClient) Shortcuts() []map[string]string {
	return []map[string]string{
		{w.buildLargeSizeShortcut: lang.Translate("mode", "Large", "stLib").String()},
		{w.buildMediumSizeShortcut: lang.Translate("mode", "Medium", "tinygo").String()},
		{w.buildSmallSizeShortcut: lang.Translate("mode", "Small", "tinygo").String()},
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

	w.storageMu.RLock()
	modeChanged := newValue != w.CurrentSizeMode
	w.storageMu.RUnlock()

	// Lazily verify TinyGo installation status ONLY when a TinyGo mode is requested
	if w.RequiresTinyGo(newValue) {
		w.verifyTinyGoInstallationStatus()
		if !w.TinyGoInstalled {
			if err := w.handleTinyGoMissing(); err != nil {
				w.Logger(err.Error())
				return
			}
			// TinyGo installed successfully — update status so builders use it
			w.TinyGoInstalled = true
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
		errorMsg := lang.Translate("Error:", "auto", "compilation", "failed:", err).String()
		//errorMsg = "Error: auto compilation failed: " + err.Error()
		w.Logger(errorMsg)
		compilationSuccess = false
		// Don't return early - still need to update assets and notify listeners
	}

	// Only notify listener when compilation succeeded and mode actually changed.
	// If compilation failed, the new mode's runtime would mismatch with the
	// old mode's .wasm binary, causing the browser to freeze on reload.
	if compilationSuccess && modeChanged && w.OnWasmExecChange != nil {
		w.OnWasmExecChange()
	}

	if compilationSuccess {
		w.LogSuccessState("Changed", "To", "Mode", newValue)
	}
}

// RecompileMainWasm recompiles the main WASM file using the current Storage mode.
func (w *WasmClient) RecompileMainWasm() error {
	w.storageMu.RLock()
	s := w.Storage
	w.storageMu.RUnlock()

	if s == nil {
		return Err("Storage not initialized")
	}

	// Use Storage.Compile() to respect In-Memory vs Disk mode
	err := s.Compile()

	if w.OnCompile != nil {
		w.OnCompile(err)
	}

	return err
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

func (w *WasmClient) storageMode() string {
	switch w.Storage.(type) {
	case *MemoryStorage:
		return "mem"
	default:
		return "disk"
	}
}

// LogSuccessState logs the standard success message with WASM details (Safe: Acquires Lock)
func (w *WasmClient) LogSuccessState(messages ...any) {
	event := lang.Translate(messages...).String()
	binarySize := "unknown"
	if sizer, ok := w.activeSizeBuilder.(interface{ BinarySize() string }); ok {
		binarySize = sizer.BinarySize()
	}
	suffix := Sprintf("[%s|%s]", w.storageMode(), binarySize)
	w.Logger(event, " ", suffix)
}
