package client

// SetMode explicitly sets the compilation mode (e.g., "S", "M", "L").
// This is useful for CLI tools or programmatic control.
func (w *WasmClient) SetMode(mode string) {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()

	// Ensure mode is valid or use default shortcuts?
	// The underlying updateCurrentBuilder handles shortcuts matching.
	w.updateCurrentBuilder(mode)

	// Also persist to DB if configured?
	if w.Database != nil {
		w.Database.Set(StoreKeySizeMode, mode)
	}
}

// Compile performs a synchronous compilation using the current settings.
// This exposes the underlying storage's Compile method.
func (w *WasmClient) Compile() error {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	if w.storage == nil {
		return nil
	}
	return w.storage.Compile()
}

// GenerateInitJS returns the JavaScript initialization code.
// This is a wrapper around GetSSRClientInitJS for external use.
func (w *WasmClient) GenerateInitJS() (string, error) {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	// Calls internal logic which uses activeSizeBuilder and current mode
	return w.GetSSRClientInitJS()
}
