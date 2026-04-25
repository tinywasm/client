# PLAN: tinywasm/client — Pending Stages

## Previous Work (completed)
> [`CHECK_PLAN.md`](CHECK_PLAN.md) — Rename `cmd/client` to `cmd/wasmbuild` + generate `script.js`. All 4 stages executed successfully.

---

## Stage 1 — Migrate TinyGo dependency to `tinywasm/tinygo` package

### Goal
Replace all internal TinyGo installation/verification logic in `tinywasm/client` with calls to the standalone `tinywasm/tinygo` package, then remove the old files.

### Dependency
`tinywasm/tinygo` package is **ready** (detect.go, install.go, download.go, extract.go, ensure.go all implemented and tested).

### Current State
- `client/tinygo_installer.go` — local installer logic (Linux-only, requires sudo for .deb)
- `client/tinygo_verify_install.go` — `VerifyTinyGoInstallation()` and `GetTinyGoVersion()` as `*WasmClient` methods
- `client/wasmbuild.go` — `runWasmBuildDeps` struct holds `ensureTinyGoInstalled` as an injected function field initialized to `EnsureTinyGoInstalled` (local package function)

### `tinywasm/tinygo` API
```go
import "github.com/tinywasm/tinygo"

tinygo.IsInstalled() bool
tinygo.GetPath() (string, error)
tinygo.GetVersion() (string, error)
tinygo.EnsureInstalled(opts ...Option) (string, error)
tinygo.Install(opts ...Option) error
```

### Files to Modify

| File | What changes |
|------|-------------|
| [go.mod](../go.mod) | Add `github.com/tinywasm/tinygo` dependency, run `go mod tidy` |
| [wasmbuild.go](../wasmbuild.go) | Replace `EnsureTinyGoInstalled()` with `tinygo.EnsureInstalled()` |
| [Change.go](../Change.go) | `verifyTinyGoInstallationStatus()` → `tinygo.IsInstalled()`, `handleTinyGoMissing()` → `tinygo.EnsureInstalled()` |
| [client.go](../client.go) | Update `WasmProjectTinyGoJsUse()`, `UseTinyGo()` internals |
| [builderInit.go](../builderInit.go) | TinyGo binary path via `tinygo.GetPath()` if applicable |

### Files to Delete

| File | Reason |
|------|--------|
| [tinygo_installer.go](../tinygo_installer.go) | All logic migrated to `tinygo` package |
| [tinygo_verify_install.go](../tinygo_verify_install.go) | Logic migrated to `tinygo/detect.go` |

### Steps
See [stages/stage1_replace_calls.md](stages/stage1_replace_calls.md) for detailed checklist.

---

## Stage 2 — Cleanup old TinyGo files and verify

### Dependency
Stage 1 completed.

### Steps
See [stages/stage2_cleanup.md](stages/stage2_cleanup.md) for detailed checklist.

---

## Stage 3 — Add `WebClientGenerator` handler (break change)

### Goal
Expose a second TUI handler from `WasmClient` that generates `web/client.go` on demand when the user clicks a button. This is an independent `HandlerExecution` — it does not affect the existing `HandlerEdit` field already provided by `WasmClient`.

### Dependency
None. Independent of Stages 1-2.

### Break Changes

**1. `CreateDefaultWasmFileClientIfNotExist` signature change**

Add a `skipIDEConfig bool` parameter. When `true`, the method skips calling `VisualStudioCodeWasmEnvConfig()`. VSCode config must only be generated once at the project root (where `go.mod` lives). When the button is triggered from a subfolder, IDE config must not be created there.

```go
// Before
func (t *WasmClient) CreateDefaultWasmFileClientIfNotExist() *WasmClient

// After (break change)
func (t *WasmClient) CreateDefaultWasmFileClientIfNotExist(skipIDEConfig bool) *WasmClient
```

Internal change: wrap `t.VisualStudioCodeWasmEnvConfig()` call with `if !skipIDEConfig`.

**2. New `webClientGenerator` type**

New unexported struct in a new file `web_client_generator.go`:

```go
type webClientGenerator struct {
    client *WasmClient
}
```

Implements `devtui.HandlerExecution`:
- `Name() string`  → returns same value as `WasmClient.Name()` for HeadlessTUI dispatch key matching
- `Label() string` → `"Generate web/client.go"`
- `Execute()`      → calls `w.client.CreateDefaultWasmFileClientIfNotExist(true)` (skipIDEConfig=true). No additional logging needed: `CreateDefaultWasmFileClientIfNotExist` already calls `t.LogSuccessState(...)` and `t.Logger(...)` internally.

Does **not** implement `devtui.Loggable`. Log output appears under WasmClient's TUI entry because `Execute()` delegates to `w.client` methods that use WasmClient's already-injected logger — not because of `Name()`.

**3. New `WebClientGenerator()` method on `WasmClient`**

```go
func (w *WasmClient) WebClientGenerator() *webClientGenerator {
    return &webClientGenerator{client: w}
}
```

Returns the handler for registration. Caller (`tinywasm/app`) invokes `AddHandler` with this value.

### Files to Create

| File | Content |
|------|---------|
| [web_client_generator.go](../web_client_generator.go) | `webClientGenerator` struct + `Name`, `Label`, `Execute` methods + `WebClientGenerator()` constructor on `WasmClient` |

### Files to Modify

| File | What changes |
|------|-------------|
| [generator.go](../generator.go) | Add `skipIDEConfig bool` parameter to `CreateDefaultWasmFileClientIfNotExist`; wrap `VisualStudioCodeWasmEnvConfig()` call with `if !skipIDEConfig` |

### Steps

- [ ] In `generator.go`: change `CreateDefaultWasmFileClientIfNotExist()` signature to `CreateDefaultWasmFileClientIfNotExist(skipIDEConfig bool)`. Wrap the `t.VisualStudioCodeWasmEnvConfig()` call (line 60) with `if !skipIDEConfig { ... }`.

- [ ] Create `web_client_generator.go`:
  - Define `webClientGenerator` struct with a single field `client *WasmClient`.
  - Implement `Name() string` — return `w.client.Name()`.
  - Implement `Label() string` — return `"Generate web/client.go"`.
  - Implement `Execute()` — call `w.client.CreateDefaultWasmFileClientIfNotExist(true)` only. Do not add extra log calls; internal logging already happens inside that method.
  - Add `WebClientGenerator() *webClientGenerator` method on `*WasmClient`.

- [ ] Fix all test call sites that pass no args to `CreateDefaultWasmFileClientIfNotExist` — add `false` (keep IDE config behavior). Affected files:
  - `tests/initialization_test.go` (1 call)
  - `tests/in_memory_test.go` (1 call)
  - `tests/generator_guard_test.go` (2 calls)
  - `tests/generator_test.go` (2 calls)

- [ ] Run `gotest` in `client/tests/` — all must pass.

- [ ] Bump module version in `go.mod` (minor version increment is correct for pre-v1 modules; this module is `v0.0.x`).
