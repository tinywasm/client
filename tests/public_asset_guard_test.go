package client_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/client"
	"github.com/tinywasm/router/mock"
)

func TestPublicAssetGuard(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock source file
	srcDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "client.go")
	err := os.WriteFile(srcFile, []byte(`package main
func main() {}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &client.Config{
		SourceDir:       func() string { return "web" },
		OutputDir:       func() string { return "web/public" },
		AssetsURLPrefix: "assets",
	}

	c := client.New(cfg)
	c.SetAppRootDir(tmpDir)
	c.SetMainInputFile("client.go")
	c.SetOutputName("test-client")

	// Helper to check routes
	checkRoutes := func(t *testing.T, r *mock.Router, storageName string) {
		routes := r.Routes()
		if len(routes) == 0 {
			t.Fatalf("[%s] No routes registered", storageName)
		}
		for _, route := range routes {
			if route.Path == "/assets/test-client.wasm" && !route.Public {
				t.Errorf("[%s] %q privada → el navegador recibe 403 y el wasm nunca carga", storageName, route.Path)
			}
		}
	}

	// 1. Check In-Memory Storage
	t.Run("MemoryStorage", func(t *testing.T) {
		r := &mock.Router{}
		c.UseMemoryStorage()
		c.RegisterRoutes(r)
		checkRoutes(t, r, "Memory")
	})

	// 2. Check Disk Storage
	t.Run("DiskStorage", func(t *testing.T) {
		r := &mock.Router{}
		c.UseDiskStorage()
		c.RegisterRoutes(r)
		checkRoutes(t, r, "Disk")
	})
}
