package client

import (
	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/fmt/lang"
	"github.com/tinywasm/tui"
)

// === DevTUI FieldHandler Interface Implementation ===

// Label returns the field label for DevTUI display
func (w *WasmClient) Label() string {
	return "Wasm Mode"
}

// Value returns the current compiler mode shortcut (c, d, or p)
func (w *WasmClient) Value() string {
	// Sync with store if available
	// w.loadMode() // REMOVED: Causes race condition/reversion if DB is stale. Mode is managed in memory via Change().

	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	// Use explicit mode tracking instead of pointer comparison
	if w.CurrentSizeMode == "" {
		return w.buildLargeSizeShortcut // Default to coding mode
	}
	return w.CurrentSizeMode
}

func (w *WasmClient) Shortcuts() []map[string]string {
	return []map[string]string{
		{w.buildLargeSizeShortcut: lang.Translate("Large", "stLib").String()},
		{w.buildMediumSizeShortcut: lang.Translate("Medium", "tinygo").String()},
		{w.buildSmallSizeShortcut: lang.Translate("Small", "tinygo").String()},
	}
}

// Options returns the compiler-mode choices as ordered {value: label} pairs so
// DevTUI renders them as a radio / segmented control (HandlerSelection). The
// data is identical to Shortcuts(): each mode's key is the value passed to
// Change() and its translated caption is the button label.
func (w *WasmClient) Options() []map[string]string {
	return w.Shortcuts()
}

// Change updates the compiler mode for WasmClient.
// Implements HandlerSelection.Change: called with the selected option's value
// ("L"/"M"/"S") when the user confirms a mode (radio) or presses a global shortcut.
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

	// Everything from here on can take a moment (TinyGo install check,
	// recompiling the wasm binary) — LogOpen/LogClose drive the TUI's
	// animated "..." indicator instead of the footer looking stuck while it
	// runs. Every exit path below must close with tui.LogClose to match.
	w.Logger(tui.LogOpen, "Compiling mode "+newValue+"...")

	// Lazily verify TinyGo installation status ONLY when a TinyGo mode is requested
	if w.RequiresTinyGo(newValue) {
		w.verifyTinyGoInstallationStatus()
		if !w.TinyGoInstalled {
			if err := w.handleTinyGoMissing(); err != nil {
				w.Logger(tui.LogClose, err.Error())
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
		w.Logger(tui.LogClose, errorMsg)
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
		event, suffix := w.buildSuccessMessage("Changed", "To", "Mode", newValue)
		w.Logger(tui.LogClose, event, " ", suffix)
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

// buildSuccessMessage formats the translated event text plus the standard
// [storage|binarySize] suffix, shared by LogSuccessState and callers that
// need to prepend their own marker (e.g. Change's tui.LogClose) to the same
// log line rather than emitting it as a separate call.
func (w *WasmClient) buildSuccessMessage(messages ...any) (event, suffix string) {
	event = lang.Translate(messages...).String()
	binarySize := "unknown"
	if sizer, ok := w.activeSizeBuilder.(interface{ BinarySize() string }); ok {
		binarySize = sizer.BinarySize()
	}
	suffix = Sprintf("[%s|%s]", w.storageMode(), binarySize)
	return event, suffix
}

// LogSuccessState logs the standard success message with WASM details (Safe: Acquires Lock)
func (w *WasmClient) LogSuccessState(messages ...any) {
	event, suffix := w.buildSuccessMessage(messages...)
	w.Logger(event, " ", suffix)
}
