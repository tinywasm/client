package client

import (
	"os"
	"strings"
	"testing"
)

// TestWasmExecContentChange verifies that the generated JavaScript content
// changes correctly when the compilation mode flag changes.
// This simulates the behavior of the external server restarting with different flags.
func TestWasmExecContentChange(t *testing.T) {
	// Helper to mimic OS args and generate JS
	generateJS := func(modeArg string) string {
		// Save original args
		origArgs := os.Args
		defer func() { os.Args = origArgs }()

		// Mock os.Args
		// The flag parsing loop looks for strings starting with "-wasmsize_mode="
		os.Args = []string{"server", "-wasmsize_mode=" + modeArg}

		js := NewJavascriptFromArgs()
		content, err := js.GetSSRClientInitJS()
		if err != nil {
			t.Fatalf("Failed to get JS for mode %s: %v", modeArg, err)
		}
		return content
	}

	// 1. Generate for Mode "L" (Go)
	contentL := generateJS("L")

	// 2. Generate for Mode "S" (TinyGo)
	contentS := generateJS("S")

	// 3. Verify they are different
	if contentL == contentS {
		t.Fatal("JS content for Mode L and Mode S is identical! It should be different (Go vs TinyGo).")
	}

	// 4. Verify Signatures
	// Go (L) usually has runtime.scheduleTimeoutEvent
	if !strings.Contains(contentL, "runtime.scheduleTimeoutEvent") {
		// Some Go versions might differ, but this is a standard verify
		t.Error("Mode L (Go) JS missing 'runtime.scheduleTimeoutEvent'")
	}

	// TinyGo (S) usually has runtime.sleepTicks
	if !strings.Contains(contentS, "runtime.sleepTicks") {
		t.Error("Mode S (TinyGo) JS missing 'runtime.sleepTicks'")
	}
}
