package client_test

import (
	"testing"

	"github.com/tinywasm/client"
	"github.com/tinywasm/tui"
)

func TestOptionsShape(t *testing.T) {
	w := client.New(nil)
	options := w.Options()
	shortcuts := w.Shortcuts()

	if len(options) != 3 {
		t.Errorf("expected 3 options, got %d", len(options))
	}

	for i := range options {
		if len(options[i]) != 1 {
			t.Errorf("option %d should have 1 entry", i)
		}
		for k, v := range options[i] {
			if sv, ok := shortcuts[i][k]; !ok || sv != v {
				t.Errorf("option %d mismatch: got %v:%v, expected match in shortcuts", i, k, v)
			}
		}
	}
}

// TestChange_LogsOpenAndClose verifies Change wraps the (potentially slow)
// compile step in tui.LogOpen/tui.LogClose so the TUI can drive its animated
// "..." indicator instead of the footer looking stuck while it compiles.
func TestChange_LogsOpenAndClose(t *testing.T) {
	w := client.New(nil)
	var logs [][]any
	w.SetLog(func(messages ...any) {
		logs = append(logs, messages)
	})

	w.Change("L")

	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log calls (open + close), got %d: %v", len(logs), logs)
	}

	first := logs[0]
	if len(first) == 0 || first[0] != tui.LogOpen {
		t.Errorf("expected first logged call to start with tui.LogOpen, got %v", first)
	}

	last := logs[len(logs)-1]
	if len(last) == 0 || last[0] != tui.LogClose {
		t.Errorf("expected last logged call to start with tui.LogClose, got %v", last)
	}
}

func TestValueOptionsConsistency(t *testing.T) {
	w := client.New(nil)
	// We use Change which might trigger compilation,
	// but since we don't have TinyGo it might fail.
	// However, Value() should still update.

	modes := []string{"L", "M", "S"}
	for _, mode := range modes {
		w.Change(mode)
		val := w.Value()
		if val != mode {
			t.Errorf("expected value %s, got %s", mode, val)
		}

		found := false
		for _, opt := range w.Options() {
			if _, ok := opt[mode]; ok {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("mode %s not found in options", mode)
		}
	}
}
