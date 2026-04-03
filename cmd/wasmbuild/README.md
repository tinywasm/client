# wasmbuild

`wasmbuild` is a CLI tool to compile Go code to WebAssembly and generate the necessary JavaScript loader.

## Installation

```bash
go install github.com/tinywasm/client/cmd/wasmbuild@latest
```

## Usage

By default, `wasmbuild` looks for `web/client.go` and compiles it to `web/public/client.wasm` using TinyGo (Mode S). It also generates `web/public/script.js` which includes `wasm_exec.js` and the initialization code.

```bash
# Default (TinyGo)
wasmbuild

# Using Standard Go compiler
wasmbuild -stdlib
```

## Requirements

- `web/client.go` file must exist.
- TinyGo must be installed (it will attempt to auto-install on Linux if missing and not using `-stdlib`).
