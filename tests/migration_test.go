package client_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLocalEmbeds(t *testing.T) {
	files, err := filepath.Glob("../*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "//go:embed assets/wasm_exec_") {
			t.Errorf("%s still contains //go:embed for wasm_exec assets", file)
		}
	}
}

func TestNoJavascriptStruct(t *testing.T) {
	forbidden := []string{
		"type Javascript ",
		"func NewJavascriptFromArgs",
		"func (w *WasmClient) GetSSRClientInitJS",
		"func (j *Javascript) GetSSRClientInitJS",
		"func WasmExecGoSignatures",
		"func WasmExecTinyGoSignatures",
		"func (w *WasmClient) WasmExecJsOutputPath",
		"func (w *WasmClient) ClearJavaScriptCache",
	}
	files, err := filepath.Glob("../*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, sym := range forbidden {
			if strings.Contains(string(content), sym) {
				t.Errorf("%s still contains removed symbol %q", file, sym)
			}
		}
	}
}
