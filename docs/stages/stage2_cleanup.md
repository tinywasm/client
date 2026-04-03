# Stage 2 — Cleanup old files and verify

### Goal
Remove migrated TinyGo files from client and verify no regressions.

### Dependency
- Stage 1 completed (all calls replaced, tests passing)

### Steps

- [ ] Delete `client/tinygo_installer.go` — all logic now in `tinygo/` package.

- [ ] Delete `client/tinygo_verify_install.go` — logic now in `tinygo/detect.go`.

- [ ] Verify no orphan imports: `go build ./...` must succeed.

- [ ] Run `gotest` in `client/tests/` — all tests must pass.

- [ ] Update `client/cmd/wasmbuild/README.md`:
  - Change "it will attempt to auto-install on Linux" to "it will attempt to auto-install on Linux, macOS, and Windows".

### Deleted Files
- `client/tinygo_installer.go`
- `client/tinygo_verify_install.go`
