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

- [ ] Define argument structures in `models.go`:
```go
package client

// SetModeArgs defines arguments for the wasm_set_mode tool.
// ormc:formonly
type SetModeArgs struct {
	Mode string `input:"required,enum=L;M;S"`
}

func (a *SetModeArgs) Schema() string { return "" }
```

## 3. Implementation Phases

### Phase 1: MCP Migration
Update the MCP tool definition to the latest protocol.

- [ ] **mcp-tool.go**: 
    - Replace `Parameters: []mcp.Parameter{...}` with `InputSchema: new(SetModeArgs).Schema()`.
    - Update `Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error)`.
    - Use `req.Bind(&args)` to decode/validate arguments.

### Phase 2: Test Reorganization
Organize all existing tests into a dedicated `tests/` directory.

- [ ] Create `tinywasm/client/tests/` directory.
- [ ] Move all `*_test.go` files from root to `tests/`.
- [ ] Update test package names to `client_test` (External testing).
- [ ] Fix imports in tests.

**Files to move:**
- `compile_modes_test.go`
- `compiler_test.go`
- `debug_test.go`
- `file_event_test.go`
- `generator_guard_test.go`
- `generator_test.go`
- `header_update_bug_test.go`
- `in_memory_test.go`
- `initialization_test.go`
- `javascripts_header_test.go`
- `javascripts_test.go`
- `output_path_test.go`
- `reproduction_test.go`
- `store_mode_test.go`
- `tinystring_test.go`
- `vscode_config_test.go`
- `wasm_exec_test.go`

## 4. Verification
- [ ] Run `ormc ./models.go`.
- [ ] Ensure all code compiles correctly.
- [ ] Run all tests from the new location: `go test ./tests/...`
