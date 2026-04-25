package client_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinywasm/client"
	"github.com/tinywasm/tinygo"
)

func TestRunWasmBuild_FailsIfInputMissing(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "wasmbuild_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temporary directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Run without web/client.go
	err = client.RunWasmBuild(client.WasmBuildArgs{Stdlib: true})
	if err == nil {
		t.Error("expected error when input file is missing, got nil")
	}
	if !strings.Contains(err.Error(), "input file not found") {
		t.Errorf("expected 'input file not found' error, got: %v", err)
	}
}

func TestRunWasmBuild_GeneratesScriptJS_TinyGo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasmbuild_test_tinygo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Create web/client.go
	if err := os.MkdirAll("web", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("web", "client.go"), []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// We only want to test script.js generation, but RunWasmBuild also compiles.
	// To avoid needing tinygo installed for this test, we might have an issue.
	// However, the plan says: "tests of generation JS no requieren compilador instalado (solo verifican el script.js). Tests de compilacion completa requieren TinyGo/Go instalado."
	// But RunWasmBuild calls Compile().

	// If we want to test ONLY JS generation, we'd need a way to skip compilation in RunWasmBuild.
	// But RunWasmBuild is monolithic as per PLAN.md.

	// Let's check if we have tinygo
	hasTinyGo := tinygo.IsInstalled()

	if !hasTinyGo {
		t.Log("Skipping TinyGo compilation part of RunWasmBuild test because TinyGo is not installed")
		// We can't easily skip just the compilation part of RunWasmBuild.
	}

	// If it fails at compilation but already wrote script.js, we can still verify script.js.
	// We use a hook to mock EnsureTinyGoInstalled so it doesn't fail even if TinyGo is missing.
	restore := client.SetRunWasmBuildHooks(client.RunWasmBuildHooks{
		EnsureTinyGoInstalled: func() (string, error) {
			return "tinygo", nil
		},
	})
	defer restore()

	_ = client.RunWasmBuild(client.WasmBuildArgs{Stdlib: false})

	scriptPath := filepath.Join("web", "public", "script.js")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("script.js was not generated: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check for TinyGo signatures
	found := false
	for _, sig := range client.WasmExecTinyGoSignatures() {
		if strings.Contains(string(content), sig) {
			found = true
			break
		}
	}
	if !found {
		t.Error("script.js does not contain TinyGo signatures")
	}

	if !strings.Contains(string(content), "instantiateStreaming") {
		t.Error("script.js does not contain instantiateStreaming")
	}
}

func TestRunWasmBuild_GeneratesScriptJS_Stdlib(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasmbuild_test_stdlib")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Create web/client.go
	if err := os.MkdirAll("web", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("web", "client.go"), []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// RunWasmBuild with Stdlib: true
	_ = client.RunWasmBuild(client.WasmBuildArgs{Stdlib: true})

	scriptPath := filepath.Join("web", "public", "script.js")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("script.js was not generated: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check for Go signatures
	found := false
	for _, sig := range client.WasmExecGoSignatures() {
		if strings.Contains(string(content), sig) {
			found = true
			break
		}
	}
	if !found {
		t.Error("script.js does not contain Go signatures")
	}
}

func TestRunWasmBuild_IncludesTinyGoEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasmbuild_env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.MkdirAll(filepath.Join("web"), 0755); err != nil {
		t.Fatal(err)
	}
	clientFile := filepath.Join("web", "client.go")
	if err := os.WriteFile(clientFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", "original-path")

	fake := &fakeRunWasmBuildClient{}
	var capturedEnv []string
	restore := client.SetRunWasmBuildHooks(client.RunWasmBuildHooks{
		EnsureTinyGoInstalled: func() (string, error) {
			return filepath.Join(tmpDir, "tinygo", "bin", "tinygo"), nil
		},
		TinyGoEnv: func() []string {
			return []string{
				"TINYGOROOT=/custom/root",
				"PATH=/custom/root/bin:/usr/bin",
			}
		},
		NewClient: func(cfg *client.Config) client.RunWasmBuildClient {
			capturedEnv = append([]string{}, cfg.Env...)
			return fake
		},
	})
	defer restore()

	if err := client.RunWasmBuild(client.WasmBuildArgs{Stdlib: false}); err != nil {
		t.Fatalf("RunWasmBuild returned error: %v", err)
	}

	if fake.compileCalls != 1 {
		t.Fatalf("expected Compile to be called once, got %d", fake.compileCalls)
	}

	if len(capturedEnv) == 0 {
		t.Fatal("expected TinyGo env to be attached to config")
	}

	foundTinyRoot := false
	for _, entry := range capturedEnv {
		if strings.HasPrefix(entry, "TINYGOROOT=") {
			foundTinyRoot = true
			if entry != "TINYGOROOT=/custom/root" {
				t.Fatalf("unexpected TINYGOROOT value: %s", entry)
			}
		}
	}
	if !foundTinyRoot {
		t.Fatal("TINYGOROOT not found in env slice")
	}

	if got := os.Getenv("PATH"); got != "original-path" {
		t.Fatalf("PATH was mutated globally: got %q", got)
	}
}

type fakeRunWasmBuildClient struct {
	compileCalls int
}

func (f *fakeRunWasmBuildClient) SetMode(string) {}

func (f *fakeRunWasmBuildClient) SetBuildOnDisk(bool, bool) {}

func (f *fakeRunWasmBuildClient) SetLog(func(...any)) {}

func (f *fakeRunWasmBuildClient) Compile() error {
	f.compileCalls++
	return nil
}

func (f *fakeRunWasmBuildClient) LogSuccessState(...any) {}
