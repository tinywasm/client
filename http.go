package client

import (
	"compress/gzip"
	"strings"

	"github.com/tinywasm/router"
)

// RegisterRoutes registers the WASM client file route on the provided router.
// It delegates to the active Storage.
func (w *WasmClient) RegisterRoutes(r router.Router) {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	w.Storage.RegisterRoutes(r)
}

func (s *MemoryStorage) RegisterRoutes(r router.Router) {
	routePath := s.Client.wasmRoutePath()

	r.PublicAsset(routePath, func(ctx router.Context) {
		s.Mu.RLock()
		content := s.WasmContent
		s.Mu.RUnlock()

		if len(content) == 0 {
			ctx.WriteStatus(503)
			ctx.Write([]byte("WASM compiling..."))
			return
		}

		ctx.SetHeader("Content-Type", "application/wasm")
		ctx.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")

		// Serve with gzip if client supports it (WASM compresses ~60-70%)
		if strings.Contains(ctx.GetHeader("Accept-Encoding"), "gzip") {
			ctx.SetHeader("Content-Encoding", "gzip")
			gz, _ := gzip.NewWriterLevel(ctx, gzip.BestCompression)
			gz.Write(content)
			gz.Close()
			return
		}

		ctx.Write(content)
	})
	s.Client.LogSuccessState("http route:", routePath)
}
