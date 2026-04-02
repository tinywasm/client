# PLAN: Refactor TinyGo dependency to use `tinywasm/tinygo` package

## Goal
Replace all internal TinyGo installation/verification logic in `tinywasm/client` with calls to the new standalone `tinywasm/tinygo` package, then remove the old code.

## Context

### Previous Plan (completed)
> [`CHECK_PLAN.md`](CHECK_PLAN.md) — Rename `cmd/client` to `cmd/wasmbuild` + generate `script.js`. All 4 stages executed successfully.

### Current State
- `client/tinygo_installer.go` — local installer logic (Linux-only, requires sudo for .deb)
- `client/tinygo_verify_install.go` — `VerifyTinyGoInstallation()` and `GetTinyGoVersion()` as `*WasmClient` methods
- `client/wasmbuild.go:19` — calls `EnsureTinyGoInstalled()` directly from client package

### Target Package
> [`tinywasm/tinygo` PLAN](../../../tinygo/docs/PLAN.md) — Standalone cross-platform installer (Linux/macOS/Windows), no sudo, tarball/zip extraction to `~/.tinywasm/tinygo/`.

**This plan corresponds to Stages 4-5 of `tinywasm/tinygo`'s PLAN.** It should only be executed after `tinygo` Stages 1-3 are completed and tested.

### New `tinygo` Package API

```go
import "github.com/tinywasm/tinygo"

tinygo.IsInstalled() bool
tinygo.GetPath() (string, error)
tinygo.GetVersion() (string, error)
tinygo.EnsureInstalled(opts ...Option) (string, error)
tinygo.Install(opts ...Option) error
```

## Files to Modify

| File | What changes |
|------|-------------|
| [go.mod](../go.mod) | Add `github.com/tinywasm/tinygo` dependency |
| [wasmbuild.go](../wasmbuild.go) | Replace `EnsureTinyGoInstalled()` with `tinygo.EnsureInstalled()` |
| [Change.go](../Change.go) | `verifyTinyGoInstallationStatus()` → `tinygo.IsInstalled()`, `handleTinyGoMissing()` → `tinygo.EnsureInstalled()` |
| [client.go](../client.go) | Update `WasmProjectTinyGoJsUse()`, `UseTinyGo()` internals |
| [builderInit.go](../builderInit.go) | TinyGo binary path via `tinygo.GetPath()` if applicable |

## Files to Delete

| File | Reason |
|------|--------|
| [tinygo_installer.go](../tinygo_installer.go) | All logic migrated to `tinygo/install.go`, `tinygo/download.go`, `tinygo/extract.go` |
| [tinygo_verify_install.go](../tinygo_verify_install.go) | Logic migrated to `tinygo/detect.go` |

## Stages

| Stage | Description | Dependency |
|-------|-------------|------------|
| 1 | [Add dependency and replace calls](stages/stage1_replace_calls.md) | `tinygo` Stages 1-3 completed |
| 2 | [Cleanup old files and verify](stages/stage2_cleanup.md) | Stage 1 |
