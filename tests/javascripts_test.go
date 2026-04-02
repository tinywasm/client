package client_test

import (
	"github.com/tinywasm/client"
	"strings"
	"testing"
)

func countSignatures(content string, sigs []string) int {
	c := 0
	for _, s := range sigs {
		if strings.Contains(content, s) {
			c++
		}
	}
	return c
}

func matchedSignatures(content string, sigs []string) []string {
	var out []string
	for _, s := range sigs {
		if strings.Contains(content, s) {
			out = append(out, s)
		}
	}
	return out
}

// TestGetSSRClientInitJSSignatures verifies that GetSSRClientInitJS()
// returns distinct JS content for Go vs TinyGo (at least one signature found each)
// and that the TinyGo usage flag changes between modes. The test is skipped
// if tinygo is not available in PATH per VerifyTinyGoInstallation requirement.
func TestGetSSRClientInitJSSignatures(t *testing.T) {
	tmpDir := t.TempDir()

	config := &client.Config{}

	// Create client.WasmClient instance with temp directory
	w := client.New(config)
	w.SetAppRootDir(tmpDir)

	// Skip only if tinygo not present
	if err := w.VerifyTinyGoInstallation(); err != nil {
		t.Skipf("skipping test: tinygo not available: %v", err)
	}

	// --- TinyGo case ---
	// Set mode to debug (requires TinyGo)
	w.CurrentSizeMode = "M"
	// ensure installer flag
	w.TinyGoInstalled = true

	// Check that WasmProjectTinyGoJsUse reports TinyGo usage
	_, tinyUsedBefore := w.WasmProjectTinyGoJsUse()
	if !tinyUsedBefore {
		t.Fatalf("expected TinyGo usage in debug mode, WasmProjectTinyGoJsUse returned false")
	}

	tinyJs, err := w.GetSSRClientInitJS()
	if err != nil {
		t.Fatalf("GetSSRClientInitJS() failed for TinyGo case: %v", err)
	}

	tinyGoFound := countSignatures(tinyJs, client.WasmExecTinyGoSignatures())
	goFoundInTiny := countSignatures(tinyJs, client.WasmExecGoSignatures())
	t.Logf("TinyGo JS matched TinyGo sigs: %v", matchedSignatures(tinyJs, client.WasmExecTinyGoSignatures()))
	t.Logf("TinyGo JS matched Go sigs: %v", matchedSignatures(tinyJs, client.WasmExecGoSignatures()))

	if tinyGoFound == 0 {
		t.Fatalf("expected at least one TinyGo signature in TinyGo JS, found none")
	}
	if goFoundInTiny != 0 {
		t.Fatalf("expected no Go signatures in TinyGo JS, but found %d", goFoundInTiny)
	}

	// --- Go case ---
	// Set mode to coding (Go standard)
	w.CurrentSizeMode = "L"

	_, tinyUsedAfter := w.WasmProjectTinyGoJsUse()
	if tinyUsedAfter {
		t.Fatalf("expected TinyGo usage to be false in coding mode, but WasmProjectTinyGoJsUse returned true")
	}

	goJs, err := w.GetSSRClientInitJS()
	if err != nil {
		t.Fatalf("GetSSRClientInitJS() failed for Go case: %v", err)
	}

	goFound := countSignatures(goJs, client.WasmExecGoSignatures())
	tinyFoundInGo := countSignatures(goJs, client.WasmExecTinyGoSignatures())
	t.Logf("Go JS matched Go sigs: %v", matchedSignatures(goJs, client.WasmExecGoSignatures()))
	t.Logf("Go JS matched TinyGo sigs: %v", matchedSignatures(goJs, client.WasmExecTinyGoSignatures()))

	if goFound == 0 {
		t.Fatalf("expected at least one Go signature in Go JS, found none")
	}
	if tinyFoundInGo != 0 {
		t.Fatalf("expected no TinyGo signatures in Go JS, but found %d", tinyFoundInGo)
	}

	// Verify the TinyGo usage flag changes between the two calls
	if tinyUsedBefore == tinyUsedAfter {
		t.Fatalf("expected TinyGo usage flag to change between debug and coding modes, but it did not")
	}
}
