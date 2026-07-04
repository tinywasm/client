package client

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/router"
)

// BuildStorage defines the behavior for compiling and serving the WASM client.
type BuildStorage interface {
	// Compile performs the compilation.
	// For Memory: compiles to buffer.
	// For Disk: compiles to disk.
	Compile() error

	// RegisterRoutes registers the WASM file handler on the router.
	RegisterRoutes(r router.Router)

	// Name returns the Storage name for logging/debugging
	Name() string
}

// MemoryStorage compiles WASM to memory and serves it directly.
type MemoryStorage struct {
	Client *WasmClient // Access to config and logger

	Mu          sync.RWMutex
	WasmContent []byte
	LastCompile time.Time
}

func (s *MemoryStorage) Name() string {
	return "In-Memory"
}

func (s *MemoryStorage) Compile() error {
	// Delegate to active builder's CompileToMemory
	// Note: activeSizeBuilder is in WasmClient
	c, ok := s.Client.activeSizeBuilder.(interface {
		CompileToMemory() ([]byte, error)
	})
	if !ok {
		return Err("active builder does not support CompileToMemory")
	}

	content, err := c.CompileToMemory()
	if err != nil {
		return err
	}

	s.Mu.Lock()
	s.WasmContent = content
	s.LastCompile = time.Now()
	s.Mu.Unlock()

	return nil
}

// DiskStorage compiles WASM to disk and serves the static file.
type DiskStorage struct {
	Client *WasmClient
}

func (s *DiskStorage) Name() string {
	return "External"
}

func (s *DiskStorage) Compile() error {
	// Ensure directory exists
	outDir := filepath.Join(s.Client.AppRootDir, s.Client.Config.OutputDir())
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Use existing CompileProgram which writes to config.OutputDir
	return s.Client.activeSizeBuilder.CompileProgram()
}

func (s *DiskStorage) RegisterRoutes(r router.Router) {
	routePath := s.Client.wasmRoutePath()
	result := filepath.Join(s.Client.Config.OutputDir(), s.Client.OutputName+".wasm")
	absPath := filepath.Join(s.Client.AppRootDir, result)

	r.Get(routePath, func(ctx router.Context) {
		content, err := os.ReadFile(absPath)
		if err != nil {
			ctx.WriteStatus(500)
			ctx.Write([]byte("Failed to read WASM file"))
			return
		}

		ctx.SetHeader("Content-Type", "application/wasm")
		ctx.Write(content)
	})
	s.Client.LogSuccessState("http route:", routePath)
}
