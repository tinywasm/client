package client

import (
	"flag"
	"os"
	"strings"
)

func init() {
	flag.String("wasmsize_mode", "", "wasm size mode (passed by tinywasm)")
}

// ParseWasmSizeModeFlag parses -wasmsize_mode flag from os.Args.
// Returns the value found, or empty string if not present.
func ParseWasmSizeModeFlag() string {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-wasmsize_mode=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}
