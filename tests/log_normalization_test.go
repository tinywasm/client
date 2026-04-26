package client_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tinywasm/client"
)

func TestLogSuccessState_Normalization(t *testing.T) {
	c := client.New(nil)

	var captured []string
	c.SetLog(func(messages ...any) {
		captured = append(captured, fmt.Sprint(messages...))
	})

	// Initial state (L mode, MemoryStorage)
	c.LogSuccessState("http route:", "/client.wasm")

	if len(captured) == 0 {
		t.Fatal("Expected log output, got none")
	}

	output := captured[0]
	t.Logf("Current output: %s", output)

	// Expected format: <event> [<mode>|<size>]
	// e.g. "http route: /client.wasm  [mem|0.0 KB]"

	if !strings.Contains(output, "http route:") {
		t.Errorf("Expected output to contain event, got: %s", output)
	}

	if !strings.Contains(output, "[mem|") {
		t.Errorf("Expected output to contain '[mem|', got: %s", output)
	}

	if strings.Contains(output, "WASMIn-Memory") {
		t.Errorf("Output still contains unnormalized 'WASMIn-Memory': %s", output)
	}
}
