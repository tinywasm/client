package client

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BuildStorage defines the behavior for compiling and serving the WASM client.
type BuildStorage interface {
	// Compile performs the compilation.
	// For Memory: compiles to buffer.
	// For Disk: compiles to disk.
	Compile() error

	// RegisterRoutes registers the WASM file handler on the mux.
	RegisterRoutes(mux *http.ServeMux)

	// Name returns the storage name for logging/debugging
	Name() string
}

// memoryStorage compiles WASM to memory and serves it directly.
type memoryStorage struct {
	client *WasmClient // Access to config and logger

	mu          sync.RWMutex
	wasmContent []byte
	lastCompile time.Time
}

func (s *memoryStorage) Name() string {
	return "In-Memory"
}

func (s *memoryStorage) Compile() error {
	s.client.Logger("Compiling WASM Client (In-Memory)...")

	// Delegate to active builder's CompileToMemory
	// Note: activeSizeBuilder is in WasmClient
	content, err := s.client.activeSizeBuilder.CompileToMemory()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.wasmContent = content
	s.lastCompile = time.Now()
	s.mu.Unlock()

	return nil
}

// diskStorage compiles WASM to disk and serves the static file.
type diskStorage struct {
	client *WasmClient
}

func (s *diskStorage) Name() string {
	return "External"
}

func (s *diskStorage) Compile() error {
	s.client.Logger("Compiling WASM Client (External/Disk)...")

	// Ensure directory exists
	outDir := filepath.Join(s.client.appRootDir, s.client.Config.OutputDir())
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Use existing CompileProgram which writes to config.OutputDir
	return s.client.activeSizeBuilder.CompileProgram()
}

func (s *diskStorage) RegisterRoutes(mux *http.ServeMux) {
	routePath := s.client.wasmRoutePath()
	result := filepath.Join(s.client.Config.OutputDir(), s.client.outputName+".wasm")
	// Note: Config.OutputDir is relative to AppRootDir usually, but ServeFile needs OS path.
	// We need absolute path.
	absPath := filepath.Join(s.client.appRootDir, result)

	mux.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, absPath)
	})
	s.client.logSuccessState("Registered External route:", routePath, "->", absPath)
}
