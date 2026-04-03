# Stage 1 — Add dependency and replace calls

### Goal
Wire `tinywasm/tinygo` package into client, replacing all local TinyGo installation/verification logic.

### Dependency
- `tinywasm/tinygo` Stages 1-3 completed (full API available and tested)

### Steps

- [ ] Add `github.com/tinywasm/tinygo` to `client/go.mod` and run `go mod tidy`.

- [ ] Update `wasmbuild.go`:
  - Replace `EnsureTinyGoInstalled()` (line 19) with `tinygo.EnsureInstalled()`.
  - Remove local import dependency on `tinygo_installer.go`.
  - Keep PATH manipulation logic using the returned path.

- [ ] Update `Change.go`:
  - `verifyTinyGoInstallationStatus()` — replace body with `w.TinyGoInstalled = tinygo.IsInstalled()`.
  - `handleTinyGoMissing()` — replace body with call to `tinygo.EnsureInstalled()`.

- [ ] Update `client.go`:
  - `VerifyTinyGoInstallation()` if still referenced — delegate to `tinygo.IsInstalled()`.
  - `GetTinyGoVersion()` if still referenced — delegate to `tinygo.GetVersion()`.
  - Keep `TinyGoInstalled` and `TinyGoCompilerFlag` fields (they are client state).

- [ ] Update `builderInit.go`:
  - If tinygo binary path is referenced, use `tinygo.GetPath()`.

- [ ] Run existing tests: `gotest` in `client/tests/` — all must pass.

### Files
- [go.mod](../../go.mod)
- [wasmbuild.go](../../wasmbuild.go)
- [Change.go](../../Change.go)
- [client.go](../../client.go)
- [builderInit.go](../../builderInit.go)
