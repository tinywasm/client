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
)

// Calling UseDiskStorage twice must leave the client in disk mode without error
// and without re-running compilation.
func TestUseDiskStorage_Idempotent(t *testing.T) {
	t.Skip("see app/docs/PLAN.md §3c — requires UseDiskStorage API")

	// TODO(agent):
	// w := client.New(...)
	// w.UseDiskStorage()
	// w.UseDiskStorage()
	// Assert: storage is *DiskStorage and no panic / no log error.
}

// Symmetric idempotency for memory mode.
func TestUseMemoryStorage_Idempotent(t *testing.T) {
	t.Skip("see app/docs/PLAN.md §3c — requires UseMemoryStorage API")

	// TODO(agent):
	// w := client.New(...)
	// w.UseMemoryStorage()
	// w.UseMemoryStorage()
	// Assert: storage is *MemoryStorage.
}

// Switching storage must NOT call Compile(). Caller composes explicitly.
// This guards against accidentally reintroducing the old `compileNow` behavior.
func TestStorageSwitch_DoesNotAutoCompile(t *testing.T) {
	t.Skip("see app/docs/PLAN.md §3c — requires pure setter semantics")

	// TODO(agent):
	// Instrument w.Compile() via a fake builder; count invocations.
	// w.UseDiskStorage()
	// w.UseMemoryStorage()
	// w.UseDiskStorage()
	// Assert compile count == 0.
}
