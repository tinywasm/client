package client_test

import (
	"testing"
	"github.com/tinywasm/client"
	"github.com/tinywasm/model"
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
