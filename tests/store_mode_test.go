package client_test

import (
	"github.com/tinywasm/client"
	"testing"
)

// MockDatabase implements KeyValueDataBase interface for testing
type MockDatabase struct {
	data map[string]string
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		data: make(map[string]string),
	}
}

func (m *MockDatabase) Get(key string) (string, error) {
	return m.data[key], nil
}

func (m *MockDatabase) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func TestValue_RespectsStoreValue(t *testing.T) {
	// 1. Setup Database with a specific mode "S" (TinyGo)
	// Default is "L" (Go)
	db := NewMockDatabase()
	db.Set(client.StoreKeySizeMode, "S")

	// 2. Initialize client.WasmClient
	cfg := client.NewConfig()
	cfg.Database = db
	// We deliberately don't set client.CurrentSizeMode here to simulate it starting with default
	// creating a fresh client that SHOULD read from store

	// Verify that WasmClient.Value() respects the value stored in the database on initialization.
	// This ensures that when the client is created, it correctly loads the previous compiler mode.

	c := client.New(cfg)

	// Verify initial state from client.New() - client.New() DOES read from store on initialization.
	if c.Value() != "S" {
		t.Fatalf("Expected initial mode 'S', got '%s'", c.Value())
	}

	// NOTE: Dynamic polling of the database in Value() is deliberately disabled to prevent
	// race conditions between local changes and stale database state.
	// The Value() method returns the current in-memory mode, which was initialized from
	// the store and is updated via Change().
}
