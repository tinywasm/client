# Plan: Migrate `client` to New `tinywasm/mcp` API

This plan outlines the steps to refactor the MCP implementation in `tinywasm/client` to comply with the updated `tinywasm/mcp` API, utilizing `ormc` for automatic JSON Schema generation and validation, and reorganizing tests.

## 1. Prerequisites & Tooling
The new API relies on `ormc` for generating schemas and validation logic from Go structs.

- [ ] Install `ormc`:
  ```bash
  go install github.com/tinywasm/orm/cmd/ormc@latest
  ```

## 2. Model Centralization (`models.go`)
Create a new file `tinywasm/client/models.go` to store all tool argument structures.

- [ ] Define argument structures in `models.go` (done).

## 3. Implementation Phases

### Phase 1: MCP Migration
Update the MCP tool definition to the latest protocol.

- [ ] **mcp-tool.go**: 
    - Replace `Parameters: []mcp.Parameter{...}` with `InputSchema: new(SetModeArgs).Schema()`.
    - Update `Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error)`.
    - Use `req.Bind(&args)` to decode/validate arguments.

### Phase 2: Test Reorganization

> [!IMPORTANT]
> **ALL 17 test files are `package client` (white-box tests).** They access private fields
> like `w.storage`, `w.tinyGoCompiler`, and helpers from `test_helpers.go`.
> They must remain `package client`, they CANNOT be changed to `package client_test`.

The correct migration strategy is:

- [ ] Create `tinywasm/client/tests/` directory.
- [ ] Rename `test_helpers.go` → `test_helpers_test.go` so the Go toolchain treats it as a
      test-only file (not compiled into the production binary). This is the key fix.
- [ ] Move all `*_test.go` files AND `test_helpers_test.go` to `tests/`.
- [ ] Keep all package declarations as `package client` (white-box, no change needed).
- [ ] Update `go.mod` or use a `tests/` internal package if needed for the build tool to find parent package symbols.

> [!WARNING]
> Moving white-box tests (`package client`) to a subdirectory means the tests are now in a
> different package path. Go requires the package they test (`package client`) to be importable.
> The `tests/` directory must import `github.com/tinywasm/client` explicitly for the non-test
> symbols, or use a build tag approach.
>
> **Recommended approach**: Keep `package client` tests alongside source in root (standard Go
> convention), OR use `package client_test` after verifying which fields can be made accessible
> via exported helpers or interfaces.

**Files to move:**
- `test_helpers.go` → rename to `test_helpers_test.go` before moving
- `compile_modes_test.go` — white-box (`package client`)
- `compiler_test.go` — white-box (`package client`)
- `debug_test.go` — white-box (`package client`)
- `file_event_test.go` — white-box (`package client`)
- `generator_guard_test.go` — white-box (`package client`)
- `generator_test.go` — white-box (`package client`)
- `header_update_bug_test.go` — white-box (`package client`)
- `in_memory_test.go` — white-box (`package client`)
- `initialization_test.go` — white-box (`package client`)
- `javascripts_header_test.go` — white-box (`package client`)
- `javascripts_test.go` — white-box (`package client`)
- `output_path_test.go` — white-box (`package client`)
- `reproduction_test.go` — white-box (`package client`)
- `store_mode_test.go` — white-box (`package client`)
- `tinystring_test.go` — white-box (`package client`)
- `vscode_config_test.go` — white-box (`package client`)
- `wasm_exec_test.go` — white-box (`package client`)

## 4. Verification
- [ ] Run `ormc` in root project.
- [ ] Ensure all code compiles correctly.
- [ ] Run all tests: `go test ./...`
