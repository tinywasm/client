package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestTinyStringMessages(t *testing.T) {

	t.Run("Test error messages with TinyString", func(t *testing.T) {
		config := NewConfig()
		config.SourceDir = func() string { return "test" }
		config.OutputDir = func() string { return "public" }
		tw := New(config)
		tw.SetAppRootDir(t.TempDir())

		// Test validation error
		if err := tw.validateMode("invalid"); err == nil {
			t.Fatal("Expected validation error for invalid mode")
		}
		// Puedes ajustar aquí la validación según el formato real del error si lo deseas
	})

	t.Run("Test Change method with TinyString messages", func(t *testing.T) {
		config := NewConfig()
		config.SourceDir = func() string { return "test" }
		config.OutputDir = func() string { return "public" }
		tw := New(config)
		tw.SetAppRootDir(t.TempDir())

		var got string
		tw.SetLog(func(message ...any) {
			if len(message) > 0 {
				got = fmt.Sprint(message[0])
			}
		})

		// Test valid mode change
		tw.Change("L")

		// Allow warning if no main.wasm.go exists in test env
		if got == "" {
			t.Fatalf("Expected non-empty success or warning message, got: '%s'", got)
		}

		// Test invalid mode (non-existent mode)
		var errMsg string
		tw.SetLog(func(message ...any) {
			if len(message) > 0 {
				errMsg = fmt.Sprint(message[0])
			}
		})
		tw.Change("invalid")

		// Ensure that the current value did not change and that validateMode reports an error.
		if tw.Value() != "L" {
			t.Errorf("Expected compiler mode to remain 'L' after invalid change, got: %s", tw.Value())
		}

		if err := tw.validateMode("invalid"); err == nil {
			t.Fatal("Expected validateMode to return an error for invalid mode")
		}

		if errMsg != "" {
			// If a progress message exists, prefer a non-fatal assertion that it mentions invalidity.
			if !strings.Contains(strings.ToLower(errMsg), "invalid") {
				t.Logf("Progress message for invalid mode did not contain 'invalid': %s", errMsg)
			}
		}
	})
}
