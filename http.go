package client

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// RegisterRoutes registers the WASM client file route on the provided mux.
// It delegates to the active Storage.
func (w *WasmClient) RegisterRoutes(mux *http.ServeMux) {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	w.Storage.RegisterRoutes(mux)
}

func (s *MemoryStorage) RegisterRoutes(mux *http.ServeMux) {
	routePath := s.Client.wasmRoutePath()

	mux.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
		s.Mu.RLock()
		content := s.WasmContent
		s.Mu.RUnlock()

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
	s.Client.LogSuccessState("http route:", routePath)
}
