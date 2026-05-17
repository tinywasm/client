# PLAN: Replace `SetBuildOnDisk(bool, bool)` with `UseDiskStorage` / `UseMemoryStorage`

> Status: Ready for execution. Breaking change. No backwards compatibility shims.
> Part of a coordinated multi-package refactor. See orchestrator:
> [tinywasm/app/docs/PLAN.md](../../app/docs/PLAN.md).

## Problem being fixed

`tinywasm/client` currently exposes:

```go
// client/client.go:186
func (w *WasmClient) SetBuildOnDisk(onDisk, compileNow bool)
```

Three defects:

### D1 — Name collision with `assetmin.SetBuildOnDisk`
The parallel refactor in [tinywasm/assetmin](../../assetmin/docs/PLAN.md) removes
its own `SetBuildOnDisk`. After that change, having a similarly-named method with
distinct semantics in `client` is a confusion magnet — readers must remember which
package they are in to interpret the call.

### D2 — Unreadable positional booleans
`SetBuildOnDisk(true, true)` carries no intent at the call site. A reader must
look up the method signature to decode what each boolean controls. Both
parameters control orthogonal concerns (storage mode vs. eager compilation).

### D3 — `onDisk=false` is never used in production
Audit of all in-tree call sites:

| Call site                              | Args         | Real intent                                  |
|----------------------------------------|--------------|----------------------------------------------|
| `app/section-build.go:77`              | `true, true` | disk + compile now (transition)              |
| `client/wasmbuild.go:133`              | `true, false`| disk; explicit `Compile()` two lines below   |
| `goflare/goflare.go:58`                | `true, false`| disk; explicit `Compile()` after             |
| `goflare/goflare.go:82`                | `true, false`| idem                                          |

`onDisk=false` appears only in tests. `compileNow=true` is a sugar that is
trivially expressible as a separate `Compile()` call. The boolean pair is
not pulling its weight.

## Breaking redesign

### 1. New API — two no-arg verbs

```go
// UseDiskStorage switches the client to disk-backed storage. Idempotent.
// Does NOT trigger compilation — the caller composes Compile() when needed.
func (w *WasmClient) UseDiskStorage()

// UseMemoryStorage switches the client to in-memory storage. Idempotent.
// Provided for symmetry and test usage; production code does not call this
// (memory is the default at construction).
func (w *WasmClient) UseMemoryStorage()
```

`Compile()` is already a public method and remains unchanged. Composition is
the responsibility of the caller.

### 2. Delete the old API

Remove `SetBuildOnDisk(onDisk, compileNow bool)` from `WasmClient`. Remove it
from the `RunWasmBuildClient` interface
([client/wasmbuild.go:14-20](../wasmbuild.go#L14-L20)):

```go
type RunWasmBuildClient interface {
    SetMode(string)
    UseDiskStorage()           // was: SetBuildOnDisk(bool, bool)
    SetLog(func(...any))
    Compile() error
    LogSuccessState(...any)
}
```

No alias, no deprecation. Forcing function for consumer migration.

### 3. Internal name collision check

`tinywasm/client` has its own `DiskStorage` / `MemoryStorage` strategy types
at [client/client.go:194-205](../client.go#L194-L205). The new public method
names (`UseDiskStorage` / `UseMemoryStorage`) mirror these internal types
intentionally — public API surface aligns with implementation vocabulary.

### 4. Call-site migration (this package's responsibility)

In-package call site to update:

```go
// client/wasmbuild.go:133
// BEFORE:
w.SetBuildOnDisk(true, false)

// AFTER:
w.UseDiskStorage()
// the existing w.Compile() at line 136 is unchanged
```

Out-of-package call sites (`app/`, `goflare/`) are migrated as part of
those packages' own work — see [app PLAN](../../app/docs/PLAN.md).

## Design rationale (alternatives considered)

**Why not an enum `SetStorage(StorageDisk)`?**
Two states do not justify a new type. The enum adds ceremony without separating
any concern that the two-method form does not already separate.

**Why not keep `compileNow` as a parameter?**
`compileNow` is literally "call `Compile()` after returning". Trivial caller
composition. Bundling it into the setter mixes two responsibilities, which is
the same antipattern this refactor removes from `assetmin` (`EnableSSRMode` vs.
`SetSSRCompiler` vs. `FlushToDisk`).

**Why keep `UseMemoryStorage` if production never calls it?**
Without it, memory mode would be an unreachable-by-name default state. Tests
need an explicit way to reset storage; future contributors need to find the
mode by symbol search; the API would be asymmetric. One line of code for
significant clarity.

**Why not just rename without splitting `compileNow`?**
That would keep `UseDiskStorage(compileNow bool)` — still one positional bool
at the call site. Not enough improvement to justify a breaking change.

## Tests

Reproducer skeleton already committed at
[../tests/storage_rename_test.go](../tests/storage_rename_test.go)
(skipped with `t.Skip` until the API exists). Coverage:

| Test                                  | What it asserts                                                              |
|---------------------------------------|------------------------------------------------------------------------------|
| `TestUseDiskStorage_Idempotent`       | Calling `UseDiskStorage` twice leaves the client in disk mode without error. |
| `TestUseMemoryStorage_Idempotent`     | Symmetric idempotency for memory mode.                                       |
| `TestStorageSwitch_DoesNotAutoCompile`| Switching storage does NOT invoke `Compile()`.                               |

Agent task: remove `t.Skip` and adapt to the implemented API. Update any
existing tests that call `SetBuildOnDisk` to use the new methods.

## Out of scope

- The `MemoryStorage` and `DiskStorage` implementations
  ([client/client.go:194-205](../client.go#L194-L205)) — unchanged.
- `Compile()` semantics — unchanged.
- WASM compilation mode (L/M/S) selection — unchanged.
- Migration of consumers in `app/` and `goflare/` — owned by those packages'
  PLANs (see [app PLAN](../../app/docs/PLAN.md)).

## Acceptance criteria

1. `SetBuildOnDisk` no longer exists on `WasmClient` (no aliases).
2. `UseDiskStorage` and `UseMemoryStorage` exist with the signatures in §1
   and are idempotent.
3. Neither method triggers `Compile()` implicitly.
4. `RunWasmBuildClient` interface is updated.
5. `client/wasmbuild.go` in-package call site is migrated.
6. The three tests above pass.
7. `go test ./...` under `client/` is green.
