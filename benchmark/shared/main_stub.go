//go:build !wasm

package main

// Stub for native toolchains (gopls, go build ./..., vet): this package is a
// wasm-only benchmark fixture — the real main lives in main.go (js && wasm)
// and is compiled by benchmark/scripts with GOOS=js GOARCH=wasm.
func main() {}
