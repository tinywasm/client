package client_test

import (
	"testing"
	"github.com/tinywasm/client"
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
