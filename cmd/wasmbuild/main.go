package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tinywasm/client"
)

func main() {
	stdlib := flag.Bool("stdlib", false, "use Go standard compiler instead of TinyGo")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Compiles web/client.go to web/public/client.wasm and generates web/public/script.js\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	err := client.RunWasmBuild(client.WasmBuildArgs{Stdlib: *stdlib})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
