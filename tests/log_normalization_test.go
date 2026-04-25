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

	// Expected normalized output: [CLIENT] <event> [<mode>|<size>]
	// e.g. [CLIENT] http route: /client.wasm [mem|0.0 KB]
	// Note: Logger([...any]) usually joins with spaces in fmt.Sprint

	if !strings.HasPrefix(output, "[CLIENT]") {
		t.Errorf("Expected output to start with '[CLIENT]', got: %s", output)
	}

	if !strings.Contains(output, "[mem|") {
		t.Errorf("Expected output to contain '[mem|', got: %s", output)
	}

	if strings.Contains(output, "WASMIn-Memory") {
		t.Errorf("Output still contains unnormalized 'WASMIn-Memory': %s", output)
	}
}
