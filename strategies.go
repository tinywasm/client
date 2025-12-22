package client

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ClientStrategy defines the behavior for compiling and serving the WASM client.
type ClientStrategy interface {
	// Compile performs the compilation.
	// For InMemory: compiles to buffer.
	// For External: compiles to disk.
	Compile() error

	// RegisterRoutes registers the WASM file handler on the mux.
	RegisterRoutes(mux *http.ServeMux)

	// Name returns the strategy name for logging/debugging
	Name() string
}

// inMemoryStrategy compiles WASM to memory and serves it directly.
type inMemoryStrategy struct {
	client *WasmClient // Access to config and logger

	mu          sync.RWMutex
	wasmContent []byte
	lastCompile time.Time
}

func (s *inMemoryStrategy) Name() string {
	return "In-Memory"
}

func (s *inMemoryStrategy) Compile() error {
	s.client.Logger("Compiling WASM Client (In-Memory)...")

	// Delegate to active builder's CompileToMemory
	// Note: activeBuilder is in WasmClient
	content, err := s.client.activeBuilder.CompileToMemory()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.wasmContent = content
	s.lastCompile = time.Now()
	s.mu.Unlock()

	return nil
}

func (s *inMemoryStrategy) RegisterRoutes(mux *http.ServeMux) {
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

// externalStrategy compiles WASM to disk and serves the static file.
type externalStrategy struct {
	client *WasmClient
}

func (s *externalStrategy) Name() string {
	return "External"
}

func (s *externalStrategy) Compile() error {
	s.client.Logger("Compiling WASM Client (External/Disk)...")

	// Ensure directory exists
	outDir := filepath.Join(s.client.appRootDir, s.client.Config.OutputDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Use existing CompileProgram which writes to config.OutputDir
	return s.client.activeBuilder.CompileProgram()
}

func (s *externalStrategy) RegisterRoutes(mux *http.ServeMux) {
	routePath := s.client.wasmRoutePath()
	fsPath := filepath.Join(s.client.Config.OutputDir, s.client.outputName+".wasm")
	// Note: Config.OutputDir is relative to AppRootDir usually, but ServeFile needs OS path.
	// We need absolute path.
	absPath := filepath.Join(s.client.appRootDir, fsPath)

	mux.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, absPath)
	})
	s.client.Logger("Registered External route:", routePath, "->", absPath)
}
