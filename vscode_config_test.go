package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVisualStudioCodeWasmEnvConfigGuard(t *testing.T) {
	tmp := t.TempDir()
	cfg := NewConfig()
	w := New(cfg)
	w.SetAppRootDir(tmp)

	t.Run("DoesNotCreateVscodeIfShouldCreateIDEConfigIsFalse", func(t *testing.T) {
		w.SetShouldCreateIDEConfig(func() bool { return false })
		w.VisualStudioCodeWasmEnvConfig()

		vscodeDir := filepath.Join(tmp, ".vscode")
		if _, err := os.Stat(vscodeDir); err == nil {
			t.Error("expected .vscode directory NOT to be created when shouldCreateIDEConfig returns false")
		}
	})

	t.Run("CreatesVscodeIfShouldCreateIDEConfigIsTrue", func(t *testing.T) {
		w.SetShouldCreateIDEConfig(func() bool { return true })
		w.VisualStudioCodeWasmEnvConfig()

		vscodeDir := filepath.Join(tmp, ".vscode")
		if _, err := os.Stat(vscodeDir); os.IsNotExist(err) {
			t.Error("expected .vscode directory to be created when shouldCreateIDEConfig returns true")
		}
	})
}
