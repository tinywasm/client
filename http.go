package client

import (
	"compress/gzip"
	"net/http"
	"strings"
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
		s.mu.RUnlock()

		if len(content) == 0 {
			http.Error(w, "WASM compiling...", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/wasm")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		// Serve with gzip if client supports it (WASM compresses ~60-70%)
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
			gz.Write(content)
			gz.Close()
			return
		}

		w.Write(content)
	})
	s.client.logSuccessState("Registered http route:", routePath)
}
