package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDefaultWasmFileClientGuard(t *testing.T) {
	tmp := t.TempDir()
	sourceDir := "web"
	cfg := NewConfig()
	cfg.SourceDir = sourceDir

	tw := New(cfg)
	tw.SetAppRootDir(tmp)
	tw.SetMainInputFile("client.go")

	t.Run("DoesNotGenerateIfGuardReturnsFalse", func(t *testing.T) {
		tw.SetShouldGenerateDefaultFile(func() bool { return false })
		tw.CreateDefaultWasmFileClientIfNotExist()

		target := filepath.Join(tmp, sourceDir, "client.go")
		if _, err := os.Stat(target); err == nil {
			t.Error("expected client.go NOT to be generated when guard returns false")
		}
	})

	t.Run("GeneratesIfGuardReturnsTrue", func(t *testing.T) {
		tw.SetShouldGenerateDefaultFile(func() bool { return true })
		tw.CreateDefaultWasmFileClientIfNotExist()

		target := filepath.Join(tmp, sourceDir, "client.go")
		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("expected client.go to be generated when guard returns true")
		}
	})
}
