package client_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/tinywasm/client"
)

type mockStorage struct {
	compileFunc func() error
}

func (m *mockStorage) Compile() error {
	if m.compileFunc != nil {
		return m.compileFunc()
	}
	return nil
}

func (m *mockStorage) RegisterRoutes(mux *http.ServeMux) {}
func (m *mockStorage) Name() string            { return "mock" }

func TestOnCompileCallback(t *testing.T) {
	w := client.New(nil)

	var called bool
	var capturedErr error

	w.SetOnCompile(func(err error) {
		called = true
		capturedErr = err
	})

	// Case 1: Success
	mockS := &mockStorage{
		compileFunc: func() error { return nil },
	}
	w.Storage = mockS

	err := w.NewFileEvent("test.go", ".go", "test.go", "write")
	if err != nil {
		t.Fatalf("NewFileEvent failed: %v", err)
	}

	if !called {
		t.Error("Expected OnCompile to be called on success")
	}
	if capturedErr != nil {
		t.Errorf("Expected nil error in OnCompile, got %v", capturedErr)
	}

	// Case 2: Failure
	called = false
	expectedErr := errors.New("compilation failed")
	mockS.compileFunc = func() error { return expectedErr }

	err = w.NewFileEvent("test.go", ".go", "test.go", "write")
	if err == nil {
		t.Error("Expected NewFileEvent to fail")
	}

	if !called {
		t.Error("Expected OnCompile to be called on failure")
	}
	if !errors.Is(capturedErr, expectedErr) {
		t.Errorf("Expected %v error in OnCompile, got %v", expectedErr, capturedErr)
	}
}
