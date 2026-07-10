package client

import (
	"github.com/tinywasm/gobuild"
)

// SetActiveBuilder sets the activeSizeBuilder.
// Useful for tests to inject a mock builder.
func (w *WasmClient) SetActiveBuilder(c gobuild.Compiler) {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()
	w.activeSizeBuilder = c
}

// SetBuilders allows injecting mock builders for all modes.
func (w *WasmClient) SetBuilders(large, medium, small gobuild.Compiler) {
	w.storageMu.Lock()
	defer w.storageMu.Unlock()
	w.builderSizeLarge = large
	w.builderSizeMedium = medium
	w.builderSizeSmall = small
}

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

// UseProductionTinyGo configures the client to compile with TinyGo in production
// (smallest possible binary). Callers (e.g. goflare) use this method instead of
// hardcoding the internal mode letter "S" — if the letter ever changes, only
// this method is updated, not every dependent.
//
// Persists the mode to disk storage so it survives across runs and is not
// silently overridden by a stale value (e.g. a previous "L" run).
func (w *WasmClient) UseProductionTinyGo() {
	w.SetMode("S")
}

// UseDebugTinyGo configures the client to compile with TinyGo in debug mode.
// Useful when iterating locally and you need TinyGo-compatible builds with
// extra debug info; never for deploy.
func (w *WasmClient) UseDebugTinyGo() {
	w.SetMode("M")
}

// UseStandardGo configures the client to compile with the standard Go compiler.
// Produces large binaries (2-10 MB) and is incompatible with edge environments
// that enforce a 1 MiB wasm limit. Useful only when binary size does not matter
// (e.g. local servers that load wasm in a desktop browser).
func (w *WasmClient) UseStandardGo() {
	w.SetMode("L")
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

// SetOnCompile. registers a callback invoked after each compilation
// triggered by a file event. err==nil indicates success.
func (w *WasmClient) SetOnCompile(fn func(err error)) {
	w.OnCompile = fn
}

// LastBuildError returns the error from the most recent compilation attempt.
func (w *WasmClient) LastBuildError() error {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	return w.lastBuildError
}
