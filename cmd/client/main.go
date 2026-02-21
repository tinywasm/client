package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinywasm/client"
)

func main() {
	// Define default paths
	defaultInput := filepath.Join("web", "client.go")
	defaultOutput := filepath.Join("web", "public", "client.wasm")

	// Parse command-line flags
	inputPtr := flag.String("input", defaultInput, "Path to the input Go file")
	outputPtr := flag.String("output", defaultOutput, "Path to the output WASM file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Compiles Go code to WebAssembly using TinyGo.\n")
		fmt.Fprintf(os.Stderr, "  Automatically installs TinyGo if missing (Linux supported).\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	inputPath := *inputPtr
	outputPath := *outputPtr

	if inputPath == "" || outputPath == "" {
		fmt.Println("Error: input and output paths cannot be empty")
		flag.Usage()
		os.Exit(1)
	}

	// Resolve absolute paths
	absInputPath, err := filepath.Abs(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving input path: %v\n", err)
		os.Exit(1)
	}
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving output path: %v\n", err)
		os.Exit(1)
	}

	// Decompose paths
	inputDir := filepath.Dir(absInputPath)
	inputFile := filepath.Base(absInputPath)

	outputDir := filepath.Dir(absOutputPath)
	outputFile := filepath.Base(absOutputPath)
	outputName := strings.TrimSuffix(outputFile, filepath.Ext(outputFile))

	// Ensure TinyGo is installed
	// We do this before setting up the client because the client builder initialization
	// might check for tinygo (though currently it just sets the command string).
	// But EnsureTinyGoInstalled returns the path to the executable, and we might need to add it to PATH.
	fmt.Println("Checking TinyGo installation...")
	tinyGoPath, err := client.EnsureTinyGoInstalled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ensuring TinyGo installation: %v\n", err)
		os.Exit(1)
	}

	if tinyGoPath != "" {
		// Add to PATH so exec.LookPath (used by gobuild or os/exec) can find it
		newPath := filepath.Dir(tinyGoPath) + string(os.PathListSeparator) + os.Getenv("PATH")
		os.Setenv("PATH", newPath)
		fmt.Printf("TinyGo found at: %s\n", tinyGoPath)
	}

	// Configure client
	cfg := client.NewConfig()
	// Use closures to return the calculated directories
	cfg.SourceDir = func() string { return inputDir }
	cfg.OutputDir = func() string { return outputDir }

	w := client.New(cfg)

	// We can leave AppRootDir as "." (default) because SourceDir/OutputDir are absolute.
	// But just to be clean, let's set it to empty so Join doesn't prepend "."
	// w.SetAppRootDir("") // New() sets it to "."

	w.SetMainInputFile(inputFile)
	w.SetOutputName(outputName)

	// Set mode to Small (TinyGo) - this uses the extension method we added
	w.SetMode("S")

	// Force external storage (disk) without immediate compile
	w.SetBuildOnDisk(true, false)

	// Trigger compilation explicitly
	fmt.Printf("Compiling %s to %s...\n", inputFile, outputFile)

	// Capture logs
	w.SetLog(func(msg ...any) {
		fmt.Println(msg...)
	})

	if err := w.Compile(); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully compiled!")
}
