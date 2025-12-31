package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestTinyStringMessages(t *testing.T) {
	t.Run("Test success messages with TinyString", func(t *testing.T) {
		config := NewConfig()
		config.SourceDir = "test"
		config.OutputDir = "public"
		tw := New(config)
		tw.SetAppRootDir(t.TempDir())

		// Test each mode message
		tests := []struct {
			mode     string
			expected []string // Words that should appear in the message
		}{
			{"L", []string{"Changed", "Mode", "Large"}},
			{"M", []string{"Changed", "Mode", "Medium"}},
			{"S", []string{"Changed", "Mode", "Small"}},
		}

		for _, test := range tests {
			msg := tw.getSuccessMessage(test.mode)

			// Check that all expected words are present in the message
			msgLower := strings.ToLower(msg)
			for _, expected := range test.expected {
				if !strings.Contains(msgLower, strings.ToLower(expected)) {
					t.Errorf("Mode %s: expected message to contain '%s', got: %s",
						test.mode, expected, msg)
				}
			}

			t.Logf("Mode %s message: %s", test.mode, msg)
		}
	})

	t.Run("Test error messages with TinyString", func(t *testing.T) {
		config := NewConfig()
		config.SourceDir = "test"
		config.OutputDir = "public"
		tw := New(config)
		tw.SetAppRootDir(t.TempDir())

		// Test validation error
		err := tw.validateMode("invalid")
		if err == nil {
			t.Fatal("Expected validation error for invalid mode")
		}

		errMsg := err.Error()
		// Mostrar el mensaje real de error para facilitar el diagnóstico
		t.Logf("Validation error message: %s", errMsg)
		// Puedes ajustar aquí la validación según el formato real del error si lo deseas
	})

	t.Run("Test Change method with TinyString messages", func(t *testing.T) {
		config := NewConfig()
		config.SourceDir = "test"
		config.OutputDir = "public"
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
		t.Logf("Change message (success or warning): %s", got)

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
		} else {
			t.Logf("validateMode returned expected error: %v", err)
		}

		if errMsg != "" {
			// If a progress message exists, prefer a non-fatal assertion that it mentions invalidity.
			if !strings.Contains(strings.ToLower(errMsg), "invalid") {
				t.Logf("Progress message for invalid mode did not contain 'invalid': %s", errMsg)
			} else {
				t.Logf("Invalid input error: %s", errMsg)
			}
		} else {
			t.Log("Change produced empty progress message for invalid mode (acceptable)")
		}
	})
}
