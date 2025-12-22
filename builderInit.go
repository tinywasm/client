package client

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/tinywasm/gobuild"
)

// builderWasmInit configures 3 builders for WASM compilation modes
func (w *WasmClient) builderWasmInit() {
	sourceDir := filepath.Join(w.appRootDir, w.Config.SourceDir)
	outputDir := filepath.Join(w.appRootDir, w.Config.OutputDir)
	mainInputFileRelativePath := filepath.Join(sourceDir, w.mainInputFile)

	// Base configuration shared by all builders
	baseConfig := gobuild.Config{
		MainInputFileRelativePath: mainInputFileRelativePath,
		OutName:                   w.outputName, // Output will be {OutputName}.wasm
		Extension:                 ".wasm",
		OutFolderRelativePath:     outputDir,
		Logger:                    w.Logger,
		Timeout:                   60 * time.Second, // 1 minute for all modes
		Callback:                  w.Callback,
	}

	// Configure Coding builder (Go standard)
	codingConfig := baseConfig
	codingConfig.Command = "go"
	codingConfig.Env = []string{"GOOS=js", "GOARCH=wasm"}
	codingConfig.CompilingArguments = func() []string {
		args := []string{"-tags", "dev"}
		if w.CompilingArguments != nil {
			args = append(args, w.CompilingArguments()...)
		}
		return args
	}
	w.builderSizeLarge = gobuild.New(&codingConfig)

	// Configure Debug builder (TinyGo debug-friendly)
	debugConfig := baseConfig
	debugConfig.Command = "tinygo"
	debugConfig.CompilingArguments = func() []string {
		args := []string{"-target", "wasm", "-opt=1"} // Keep debug symbols
		if w.CompilingArguments != nil {
			args = append(args, w.CompilingArguments()...)
		}
		return args
	}
	w.builderSizeMedium = gobuild.New(&debugConfig)

	// Configure Production builder (TinyGo optimized)
	prodConfig := baseConfig
	prodConfig.Command = "tinygo"
	prodConfig.CompilingArguments = func() []string {
		args := []string{"-target", "wasm", "-opt=z", "-no-debug", "-panic=trap"}
		if w.CompilingArguments != nil {
			args = append(args, w.CompilingArguments()...)
		}
		return args
	}
	w.builderSizeSmall = gobuild.New(&prodConfig)

	// Set initial mode and active builder (default to coding mode)
	w.activeSizeBuilder = w.builderSizeLarge // Default: fast development
}

// updateCurrentBuilder sets the activeSizeBuilder based on mode and cancels ongoing operations
func (w *WasmClient) updateCurrentBuilder(mode string) {
	// 1. Cancel any ongoing compilation
	if w.activeSizeBuilder != nil {
		w.activeSizeBuilder.Cancel()
	}

	// 2. Update current mode tracking
	w.currenSizeMode = mode

	// 3. Set activeSizeBuilder based on mode
	switch mode {
	case w.buildLargeSizeShortcut: // "L"
		w.activeSizeBuilder = w.builderSizeLarge
	case w.buildMediumSizeShortcut: // "M"
		w.activeSizeBuilder = w.builderSizeMedium
	case w.buildSmallSizeShortcut: // "S"
		w.activeSizeBuilder = w.builderSizeSmall
	default:
		w.activeSizeBuilder = w.builderSizeLarge // fallback to coding mode
	}
}

// OutputRelativePath returns the RELATIVE path to the final output file
// eg: "deploy/edgeworker/app.wasm" (relative to AppRootDir)
// This is used by file watchers to identify output files that should be ignored.
// The returned path always uses forward slashes (/) for consistency across platforms.
func (w *WasmClient) OutputRelativePath() string {
	// FinalOutputPath() returns absolute path like: /tmp/test/deploy/edgeworker/app.wasm
	// We need to extract the relative portion: deploy/edgeworker/app.wasm
	fullPath := w.activeSizeBuilder.FinalOutputPath()

	// Remove AppRootDir prefix to get relative path
	if strings.HasPrefix(fullPath, w.appRootDir) {
		relPath := strings.TrimPrefix(fullPath, w.appRootDir)
		// Remove leading separator (/ or \)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		relPath = strings.TrimPrefix(relPath, "/")  // Handle Unix paths
		relPath = strings.TrimPrefix(relPath, "\\") // Handle Windows paths
		// Normalize to forward slashes for consistency (replace all backslashes)
		return strings.ReplaceAll(relPath, "\\", "/")
	}

	// Fallback: construct from config values (which are already relative)
	// Normalize to forward slashes for consistency
	result := filepath.Join(w.Config.OutputDir, w.outputName+".wasm")
	return strings.ReplaceAll(result, "\\", "/")
}
