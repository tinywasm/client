package client

import (
	"testing"
)

// MockStore implements Store interface for testing
type MockStore struct {
	data map[string]string
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]string),
	}
}

func (m *MockStore) Get(key string) (string, error) {
	return m.data[key], nil
}

func (m *MockStore) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func TestJavascriptForInitializing_RespectsStoreValue(t *testing.T) {
	// 1. Setup Store with a specific mode "S" (TinyGo)
	// Default is "L" (Go)
	store := NewMockStore()
	store.Set(StoreKeyBuildMode, "S")

	// 2. Initialize WasmClient
	cfg := NewConfig()
	cfg.Store = store
	// We deliberately don't set currentMode here to simulate it starting with default
	// creating a fresh client that SHOULD read from store

	// BUT, the user says "when ANOTHER handler calls JavascriptForInitializing".
	// This implies the client might have been initialized ALREADY, and THEN the store changes?
	// Or maybe the client is initialized, and then we expect it to read from the store on every call?

	// If the user says "cuando otro manejador llama a ... este no esta respetando el valor que esta almacenado en Store"
	// it likely means they expect dynamic updates from the store.

	client := New(cfg)

	// Verify initial state from New() - New() DOES read from store on initialization.
	if client.Value() != "S" {
		t.Fatalf("Expected initial mode 'S', got '%s'", client.Value())
	}

	// Now, let's simulate the store changing externally (or just being different from what the client thinks if it wasn't refreshed)
	// Or maybe the user means: I have a client, I change the store via some other means, and report back.

	store.Set(StoreKeyBuildMode, "L") // Change back to L in store

	// Client.Value() currently caches the value in w.currentMode.
	// If Value() doesn't check the store, it will still return "S" (from initialization).

	mode := client.Value()
	if mode != "L" {
		t.Fatalf("Bug replicated: Expected mode 'L' from store, but got cached '%s'", mode)
	}
}
