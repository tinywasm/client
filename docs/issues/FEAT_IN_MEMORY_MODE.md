# Feature: In-Memory WASM Client Mode

**Objective**: Refactor `client` (WasmClient) to support an "In-Memory" mode where the WASM binary is compiled to RAM and served directly, similar to the `server` package refactor.

## Context
The `client` package currently compiles `.wasm` files to disk. To reduce filesystem clutter during development, we want to default to compiling in-memory and only write to disk when explicitly requested (e.g., "Eject" or "Create Template").

## Requirements

1.  **Configuration**:
    *   Add `AssetsURLPrefix` to `Config` (default `""`). This prefixes the served URL (e.g., `assets/client.wasm`).
    *   Preserve existing `OutputDir` for External mode.

2.  **Strategies**:
    *   Implement **Strategy Pattern** (similar to `server` package):
        *   `InMemoryStrategy`:
            *   Uses `gobuild.CompileToMemory()`.
            *   Stores `[]byte` of the compiled WASM.
            *   Serves via `RegisterRoutes` at `/{AssetsURLPrefix}/{OutputName}.wasm` with `Content-Type: application/wasm`.
        *   `ExternalStrategy`:
            *   Uses existing `gobuild.CompileProgram()` (to disk).
            *   Serves via `RegisterRoutes` (using `http.ServeFile`).

3.  **Routing**:
    *   Add `RegisterRoutes(mux *http.ServeMux)` method to `WasmClient`.
    *   This method delegates to the active strategy to register the WASM file route.

4.  **Lifecycle**:
    *   **Startup**: If `{OutputDir}/{OutputName}.wasm` exists -> **External Mode**.
    *   **Startup**: If file missing -> **In-Memory Mode**.
    *   **Events**:
        *   On `.go` file change: Recompile (to RAM or Disk based on mode). call `OnWasmExecChange` or similar callback to notify app (trigger browser reload).

5.  **Switching**:
    *   Add method `CreateDefaultWasmFileClientIfNotExist` (or similar rename) that forces transition to **External Mode**.

6.  **Refactoring**:
    *   Update `New` to select strategy.
    *   Update `NewFileEvent` to delegate to strategy.

## Verification
*   Test In-Memory mode serves valid WASM.
*   Test External mode writes to disk.
*   Test switching modes.
