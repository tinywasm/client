> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Part of the orchestrator: `../../docs/HOT_RELOAD_MASTER_PLAN.md` (Phase D). Depends on `gobuild/docs/PLAN.md` (Phase A) being merged first — this plan imports `gobuild.Compiler`.

# client — consume gobuild.Compiler interface, add fast reload-path tests

## Problem

`WasmClient` (`client/client.go:19-22`) holds 4 builder fields typed as
concrete `*gobuild.GoBuild` (`builderSizeLarge`, `builderSizeMedium`,
`builderSizeSmall`, `activeSizeBuilder`). Every test that exercises
`NewFileEvent` → `Storage.Compile()` → real builder either mocks at the
`Storage` interface boundary (`MemoryStorage`/`DiskStorage`, see
`client/storage.go`) or runs a real `go build`/tinygo. There is no test
proving `SetOnWasmExecChange` fires only on an actual mode-change tied to a
real successful compile — this is one of the 3 uncoordinated reload paths
identified in the master plan.

## Required change

1. Change the 4 builder fields' type from `*gobuild.GoBuild` to
   `gobuild.Compiler` (the interface added in the `gobuild` phase). This is
   the breaking change: anything constructing a `WasmClient` and reaching
   into these fields directly (check `app/section-build.go` and any test
   helpers) must be updated to accept the interface type.
2. `builderWasmInit()` (referenced at `client.go:90`, defined elsewhere —
   locate it) must keep constructing real `*gobuild.GoBuild` instances by
   default (`gobuild.New(...)` already returns something satisfying
   `Compiler`) — no behavior change for production wiring.
3. Add a constructor variant or setter allowing tests to inject
   `gobuild.FakeCompiler` (from the `gobuild` mock package) as
   `activeSizeBuilder` without needing a real `SourceDir`/`OutputDir` on
   disk. Name it explicitly, e.g. `func (w *WasmClient) SetActiveBuilder(c gobuild.Compiler)` if it doesn't already exist in some form — check `client/Change.go` first since `RecompileMainWasm` (line 72) likely already swaps `activeSizeBuilder`.
4. Do **not** change `Storage`/`BuildStorage`'s existing interface — it's
   already an abstraction; this change is only about the builder fields
   underneath the storage implementations.

## Reload-path test gap to close

Per the master plan's root-cause analysis, `SetOnWasmExecChange`
(`client/file_event.go` — confirm exact call site, the master plan
investigation places it around `OnWasmExecChange` invocation gated on
compile success at `file_event.go:16`) must **only** fire its callback when:
   a. A compile was actually triggered (not on unrelated file events).
   b. The compile succeeded.
   c. The wasm exec runtime actually changed mode (not on every compile).

Add `client/tests/on_wasm_exec_change_test.go` using `FakeCompiler`:

1. Inject a `FakeCompiler` with `CompileErr = someErr`. Trigger
   `NewFileEvent`. Assert `OnWasmExecChange` callback is **not** called.
2. Inject a `FakeCompiler` with `CompileErr = nil` but no mode change.
   Trigger `NewFileEvent`. Assert callback **not** called.
3. Force an actual mode switch (`RecompileMainWasm`/`Change`, see
   `client/Change.go:18,72`) with `FakeCompiler.CompileErr = nil`. Assert
   `OnWasmExecChange` **is** called exactly once.
4. Assert `OnCompile(err)` is called with the exact error/nil in all 3
   cases above, independent of `OnWasmExecChange` — the two callbacks are
   not the same event and must not be conflated.

## Constraints

- No hardcoded strings — reuse existing mode constants
  (`buildLargeSizeShortcut` etc., already named) rather than introducing new
  literals.
- Must not break existing `client/tests/*` — run `gotest ./...` in
  `client/` after the change; existing `file_event_test.go`,
  `on_compile_test.go`, `compiler_test.go`, `wasmbuild_test.go`,
  `in_memory_test.go`, `store_mode_test.go` must still pass.
- Value embedding / TinyGo constraints from `core-principles` still apply if
  this package is ever wasm-compiled itself — confirm `client` is a
  server-side-only (non-wasm) package before assuming this doesn't matter;
  if it's never compiled to wasm itself, this constraint is moot for this
  plan.

## Stages

| Stage | Description | Output |
|---|---|---|
| 1 | Change 4 builder fields to `gobuild.Compiler` type; fix all call sites | `client/client.go`, `client/Change.go`, `client/client_extensions.go` compiling clean |
| 2 | Add test-only injection point for a fake active builder | New exported setter or confirm existing one suffices |
| 3 | Add `client/tests/on_wasm_exec_change_test.go` with the 4 assertions above | New test file, passing |
| 4 | Run full `client` test suite, confirm no regressions | Test output attached to PR |
