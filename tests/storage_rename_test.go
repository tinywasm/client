package client_test

// Reproducer test suite for the storage API rename
// (see tinywasm/app/docs/PLAN.md §3c).
//
// Skipped today because UseDiskStorage / UseMemoryStorage do not yet exist.
// The agent implementing the PLAN MUST remove t.Skip and adapt.
//
// Rename:
//   SetBuildOnDisk(onDisk, compileNow bool)
//     → UseDiskStorage()  (no auto-compile; caller invokes Compile() explicitly)
//     → UseMemoryStorage()

import (
	"testing"

	"github.com/tinywasm/client"
)

// Calling UseDiskStorage twice must leave the client in disk mode without error
// and without re-running compilation.
func TestUseDiskStorage_Idempotent(t *testing.T) {
	w := client.New(nil)
	w.UseDiskStorage()
	w.UseDiskStorage()

	if name := w.Storage.Name(); name != "External" {
		t.Errorf("Expected External storage, got %s", name)
	}
}

// Symmetric idempotency for memory mode.
func TestUseMemoryStorage_Idempotent(t *testing.T) {
	w := client.New(nil)
	w.UseDiskStorage()   // Start on disk
	w.UseMemoryStorage()
	w.UseMemoryStorage()

	if name := w.Storage.Name(); name != "In-Memory" {
		t.Errorf("Expected In-Memory storage, got %s", name)
	}
}

// Switching storage must NOT call Compile(). Caller composes explicitly.
// This guards against accidentally reintroducing the old `compileNow` behavior.
func TestStorageSwitch_DoesNotAutoCompile(t *testing.T) {
	w := client.New(nil)

	compileCount := 0
	w.SetOnCompile(func(err error) {
		compileCount++
	})

	w.UseDiskStorage()
	w.UseMemoryStorage()
	w.UseDiskStorage()

	if compileCount != 0 {
		t.Errorf("Expected 0 compilations, got %d", compileCount)
	}
}
