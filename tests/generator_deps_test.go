package client_test

import (
	"github.com/tinywasm/client"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDefaultWasmFileClientEnsuresDependencies(t *testing.T) {
	// Skip if no 'go' command
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not found")
	}

	tmp := t.TempDir()

	// Initialize a go mod in the tmp dir
	cmd := exec.Command("go", "mod", "init", "testmod")
	cmd.Dir = tmp
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init go mod: %v", err)
	}

	sourceDir := "web"
	cfg := client.NewConfig()
	cfg.SourceDir = func() string { return sourceDir }

	tw := client.New(cfg)
	tw.SetAppRootDir(tmp)
	tw.SetMainInputFile("client.go")
	tw.SetShouldGenerateDefaultFile(func() bool { return true })

	// We need to mock Compile to avoid actual compilation if tinygo is missing
	// but we want to see if go.mod is updated.
	// Actually, generator.go calls store.Compile().
	// We can use a custom storage.
	tw.UseMemoryStorage()

	// Run generation
	tw.CreateDefaultWasmFileClientIfNotExist(true)

	// Verify go.mod was updated
	goModPath := filepath.Join(tmp, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	goModContent := string(content)
	expectedDeps := []string{
		"github.com/tinywasm/dom",
		"github.com/tinywasm/fmt",
		"github.com/tinywasm/html",
	}

	for _, dep := range expectedDeps {
		if !strings.Contains(goModContent, dep) {
			t.Errorf("go.mod missing dependency: %s", dep)
		}
	}
}
