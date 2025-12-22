package client

import (
	"os/exec"
	"testing"
)

// TestStoreRoundtrip ensures the mode is saved to and loaded from the Store
func TestStoreRoundtrip(t *testing.T) {
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Skip("tinygo not found in PATH")
	}

	store := &testDatabase{data: make(map[string]string)}

	config := &Config{
		Logger:   func(...any) {},
		Database: store,
	}

	New(config)

	// Test all three supported shortcuts: coding, debugging, production
	shortcuts := []string{"L", "M", "S"}

	for _, mode := range shortcuts {
		// Use a fresh WasmClient instance per mode to avoid shared state
		w := New(config)

		progress := make(chan string, 10)
		w.Change(mode, progress)
		close(progress) // Close the channel since Change doesn't

		// Check that mode is saved in store
		saved, err := store.Get(StoreKeySizeMode)
		if err != nil {
			t.Fatalf("failed to get mode from store for %q: %v", mode, err)
		}
		if saved != mode {
			t.Fatalf("expected saved mode %q, got %q", mode, saved)
		}

		// Create new instance to test loading
		w2 := New(config)
		if w2.Value() != mode {
			t.Fatalf("expected loaded mode %q, got %q", mode, w2.Value())
		}
	}
}
