# tinywasm/client
<img src="docs/img/badges.svg">

**Build-only** WebAssembly compilation manager for TinyWASM.

> As of v2, `client` has a single responsibility: **compile the WASM binary** and serve it at `/client.wasm`. All JavaScript composition has moved to [`tinywasm/js`](https://github.com/tinywasm/js).

## Responsibilities

| ✅ Owns | ❌ No longer owns |
|---|---|
| WASM compilation (Go stdlib / TinyGo) | `wasm_exec.js` embedding |
| Serving `/client.wasm` via HTTP | `Javascript` struct / `GetSSRClientInitJS` |
| 3-mode compiler selection (L/M/S) | Generating `script.js` content |
| File watcher for `web/main.wasm.go` | `WasmExecGoSignatures` / `WasmExecTinyGoSignatures` |
| VS Code GOOS/GOARCH config | `ClearJavaScriptCache` |
| Project scaffolding (`CreateDefaultWasmFileClientIfNotExist`) | |

## CLI Tool

```bash
go install github.com/tinywasm/client/cmd/wasmbuild@latest
wasmbuild        # Go stdlib mode (L)  → writes web/public/client.wasm + script.js
wasmbuild -tinygo # TinyGo mode (S)
```

`script.js` is now generated via `tinywasm/js.PageBootstrap()` — the embedded `wasm_exec.js` lives in `js/assets/` (single source of truth).

See [cmd/wasmbuild](cmd/wasmbuild/README.md) for details.

## 🛠 Basic Usage

```go
cfg := client.NewConfig()
cfg.SourceDir = func() string { return "web" }
cfg.OutputDir = func() string { return "web/public" }

twc := client.New(cfg)
twc.SetAppRootDir("/path/to/project")
twc.SetMainInputFile("app.go") // default: "client.go"
twc.SetOutputName("app")       // default: "client"
```

### Mode switching

```go
twc.Change("S") // "L" = Go, "M" = TinyGo debug, "S" = TinyGo prod
```

### HTTP serving

```go
twc.RegisterRoutes(mux) // registers /client.wasm (or /prefix/client.wasm)
```

### ArgumentsForServer

```go
// Returns []string{"-wasmsize_mode=L"} (for subprocess injection)
args := twc.ArgumentsForServer()
```

## Storage modes

| Mode | Use case |
|---|---|
| In-Memory (default) | Fast dev — compiles to buffer, served directly |
| Disk | Static integration — compiles to `OutputDir`, served via `http.ServeFile` |

```go
twc.UseDiskStorage()   // switch to disk
twc.UseMemoryStorage() // switch back to memory
```

## Project Initialization

```go
// Generates web/client.go + .vscode/settings.json if missing.
// Automatically adds required modules (dom, html, fmt) to go.mod.
twc.CreateDefaultWasmFileClientIfNotExist(false)
```

## ⚙️ Configuration

- **`Config` struct**: shared deps (Store, Logger), directory functions (`SourceDir`, `OutputDir`). See [config.go](config.go).
- **Setters**: reactive — re-initialize internal state automatically.

## 📋 Requirements

- Go 1.21+
- [TinyGo](https://tinygo.org/) (optional — needed for M/S modes only)

## JavaScript composition

All JS is now the responsibility of `tinywasm/js`:

```go
import "github.com/tinywasm/js"

// In tinywasm/app boot:
js.SetRuntime(js.RuntimeGo) // or RuntimeTinyGo
scripts := []any{js.PageBootstrap()} // registers with assetmin
```

See [`tinywasm/js`](https://github.com/tinywasm/js) for the full typed API including `ServiceWorker` and `WebWorker`.
