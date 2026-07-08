package client_test

import (
	"github.com/tinywasm/client"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/model"
	"testing"
)

func TestSetModeArgsValidation(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"L", false},
		{"M", false},
		{"S", false},
		{"X", true},
		{"LM", true},
		{"", true},
	}

	for _, tt := range tests {
		args := &client.SetModeArgs{Mode: tt.mode}
		err := model.ValidateFields('u', args)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateFields('u', {Mode: %q}) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
		}
	}
}

// TestSetModeArgsBind proves the wasm_set_mode tool path end to end at the
// binding layer: req.Bind decodes the JSON arguments AND runs the generated
// Validate, so an out-of-enum mode never reaches WasmClient.Change.
func TestSetModeArgsBind(t *testing.T) {
	bind := func(argsJSON string) error {
		req := mcp.Request{
			Params: mcp.CallToolParams{Name: "wasm_set_mode", Arguments: argsJSON},
			Action: 'u',
		}
		var args client.SetModeArgs
		return req.Bind(&args)
	}

	if err := bind(`{"mode":"M"}`); err != nil {
		t.Errorf(`Bind({"mode":"M"}) should succeed, got: %v`, err)
	}
	if err := bind(`{"mode":"X"}`); err == nil {
		t.Error(`Bind({"mode":"X"}) should fail (mode outside L/M/S)`)
	}
	if err := bind(`{}`); err == nil {
		t.Error(`Bind({}) should fail (mode is NotNull)`)
	}
}
