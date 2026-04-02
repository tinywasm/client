package client_test

import (
	"github.com/tinywasm/client"
	"os/exec"
	"testing"
)

// TestStoreModePersistence tests that mode changes are correctly saved to the Store
// and persist across client.WasmClient instances, simulating the bug where mode updates were not persisted.
func TestStoreModePersistence(t *testing.T) {
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Skip("tinygo not found in PATH")
	}

	store := &TestDatabase{data: make(map[string]string)}

	config := client.NewConfig()
	config.Database = store

	// Step 1: Start with initial mode (should be L)
	w1 := client.New(config)
	w1.SetAppRootDir(t.TempDir())
	if w1.Value() != "L" {
		t.Errorf("Initial mode should be 'L', got '%s'", w1.Value())
	}

	// Step 2: Change to Medium mode
	w1.Change("M")

	saved, _ := store.Get(client.StoreKeySizeMode)
	if saved != "M" {
		t.Errorf("After changing to 'M', store should have 'M', got '%s'", saved)
	}

	// Verify new instance loads the mode
	w2 := client.New(config)
	if w2.Value() != "M" {
		t.Errorf("client.New instance should load 'M', got '%s'", w2.Value())
	}

	// Step 3: Change to Small mode (critical test for the bug)
	w2.Change("S")

	saved, _ = store.Get(client.StoreKeySizeMode)
	if saved != "S" {
		t.Errorf("After changing to 'S', store should have 'S', got '%s'", saved)
	}

	w3 := client.New(config)
	if w3.Value() != "S" {
		t.Errorf("client.New instance should load 'S', got '%s'", w3.Value())
	}

	// Step 4: Test back and forth to ensure robustness
	modes := []string{"M", "S", "L", "M", "S"}
	for _, mode := range modes {
		w := client.New(config)
		w.Change(mode)

		saved, _ := store.Get(client.StoreKeySizeMode)
		if saved != mode {
			t.Errorf("After changing to '%s', store should have '%s', got '%s'", mode, mode, saved)
		}

		wNew := client.New(config)
		if wNew.Value() != mode {
			t.Errorf("client.New instance should load '%s', got '%s'", mode, wNew.Value())
		}
	}
}
