package client

// SetMode explicitly sets the compilation mode (e.g., "S", "M", "L").
// This is useful for CLI tools or programmatic control.
func (w *WasmClient) SetMode(mode string) {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()

	// Ensure mode is valid or use default shortcuts?
	// The underlying UpdateCurrentBuilder handles shortcuts matching.
	w.UpdateCurrentBuilder(mode)

	// Also persist to DB if configured?
	if w.Database != nil {
		w.Database.Set(StoreKeySizeMode, mode)
	}
}

// Compile performs a synchronous compilation using the current settings.
// This exposes the underlying Storage's Compile method.
func (w *WasmClient) Compile() error {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()

	if w.Storage == nil {
		return nil
	}
	return w.Storage.Compile()
}
