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
		fmt.Fprintf(os.Stderr, "  Automatically installs TinyGo if missing (Linux supported).\n")
		fmt.Fprintf(os.Stderr, "  Generates script.js in output directory if missing.\n\n")
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
	cfg.SourceDir = func() string { return inputDir }
	cfg.OutputDir = func() string { return outputDir }

	w := client.New(cfg)
	w.SetMainInputFile(inputFile)
	w.SetOutputName(outputName)

	// Set mode to Small (TinyGo)
	w.SetMode("S")

	// Force external storage (disk) without immediate compile
	w.SetBuildOnDisk(true, false)

	// Trigger compilation explicitly
	fmt.Printf("Compiling %s to %s...\n", inputFile, outputFile)

	w.SetLog(func(msg ...any) {
		fmt.Println(msg...)
	})

	if err := w.Compile(); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Check and generate script.js if missing
	scriptJsPath := filepath.Join(outputDir, "script.js")
	if _, err := os.Stat(scriptJsPath); os.IsNotExist(err) {
		fmt.Printf("script.js not found at %s, generating...\n", scriptJsPath)

		jsContent, err := w.GenerateInitJS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate JS content: %v\n", err)
			// Don't exit, as WASM might still be usable manually
		} else {
			if err := os.WriteFile(scriptJsPath, []byte(jsContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write script.js: %v\n", err)
			} else {
				fmt.Printf("Generated %s\n", scriptJsPath)
			}
		}
	} else {
		fmt.Printf("script.js already exists at %s, skipping generation.\n", scriptJsPath)
	}

	fmt.Println("Successfully compiled!")
}
