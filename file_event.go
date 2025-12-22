package client

import (
	. "github.com/tinywasm/fmt"
)

func (w *WasmClient) SupportedExtensions() []string {
	return []string{".go"}
}

// NewFileEvent handles file events for WASM compilation with automatic project detection
// fileName: name of the file (e.g., main.wasm.go)
// extension: file extension (e.g., .go)
// filePath: full path to the file (e.g., ./home/userName/ProjectName/web/public/main.wasm.go)
// event: type of file event (e.g., create, remove, write, rename)
func (w *WasmClient) NewFileEvent(fileName, extension, filePath, event string) error {
	const e = "NewFileEvent Wasm"

	if filePath == "" {
		return Err(e, "filePath is empty")
	}

	w.Logger(extension, event, "...", filePath)

	// Only process Go files for compilation triggers
	if extension != ".go" {
		return nil
	}

	// Only process write/create events
	if event != "write" && event != "create" {
		return nil
	}

	// IMPORTANT: At this point, devwatch has already called depfind.ThisFileIsMine()
	// and confirmed this file belongs to this handler. We should ALWAYS compile.
	// The old ShouldCompileToWasm() check was incorrect - it rejected dependency files.

	// Compile using current storage (In-Memory or External)
	if w.storage == nil {
		return Err("storage not initialized")
	}

	w.Logger("Compiling WASM due to", filePath, "change...")

	// Compile using storage
	if err := w.storage.Compile(); err != nil {
		return Err("compiling to WebAssembly error: ", err)
	}

	w.Logger("✓ WASM compilation successful")

	if w.OnWasmExecChange != nil {
		w.OnWasmExecChange()
	}

	return nil
}

// ShouldCompileToWasm determines if a file should trigger WASM compilation
func (w *WasmClient) ShouldCompileToWasm(fileName, filePath string) bool {
	// Always compile main.wasm.go
	if fileName == w.mainInputFile {
		return true
	}

	// Any .wasm.go file should trigger compilation
	if HasSuffix(fileName, ".wasm.go") {
		return true
	}

	// All other files should be ignored
	return false
}

// MainInputFileRelativePath returns the relative path to the main WASM input file (e.g. "main.wasm.go").
func (w *WasmClient) MainInputFileRelativePath() string {
	// The input lives under the source directory by convention.
	// Return full path including AppRootDir for callers that expect absolute paths
	return PathJoin(w.Config.SourceDir, w.mainInputFile).String()
}

// MainOutputFileAbsolutePath returns the absolute path to the main WASM output file (e.g. "main.wasm").
func (w *WasmClient) MainOutputFileAbsolutePath() string {
	// The output file is created in OutputDir which is:
	// AppRootDir/OutputDir/{OutputName}.wasm
	return PathJoin(w.appRootDir, w.Config.OutputDir, w.outputName+".wasm").String()
}

// UnobservedFiles returns files that should not be watched for changes e.g: main.wasm
func (w *WasmClient) UnobservedFiles() []string {
	return w.activeSizeBuilder.UnobservedFiles()
}
