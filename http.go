package client

import (
	"bytes"
	"net/http"
)

// RegisterRoutes registers the WASM client file route on the provided mux.
// It delegates to the active storage.
func (w *WasmClient) RegisterRoutes(mux *http.ServeMux) {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	w.storage.RegisterRoutes(mux)
}

func (s *memoryStorage) RegisterRoutes(mux *http.ServeMux) {
	routePath := s.client.wasmRoutePath()

	mux.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		content := s.wasmContent
		lastMod := s.lastCompile
		s.mu.RUnlock()

		if len(content) == 0 {
			// If not yet compiled, try to compile on demand (lazy loading)
			// But careful with concurrency. For now, just error or wait.
			// Let's try to trigger a compile if empty? Or just return 503.
			http.Error(w, "WASM compiling...", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/wasm")
		http.ServeContent(w, r, s.client.outputName+".wasm", lastMod, bytes.NewReader(content))
	})
	s.client.Logger("Registered In-Memory route:", routePath)
}
