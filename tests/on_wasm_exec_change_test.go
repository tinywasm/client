package client_test

import (
	"errors"
	"testing"

	"github.com/tinywasm/client"
	"github.com/tinywasm/gobuild/mock"
)

type fakeCompiler struct {
	*gobuildmock.FakeCompiler
}

func (f *fakeCompiler) CompileToMemory() ([]byte, error) {
	f.CompileCallCount++
	return []byte(f.Output), f.CompileErr
}

func (f *fakeCompiler) Cancel() error {
	return nil
}

func (f *fakeCompiler) BinarySize() string {
	return "0 B"
}

func newFakeCompiler() *fakeCompiler {
	return &fakeCompiler{
		FakeCompiler: &gobuildmock.FakeCompiler{},
	}
}

func TestOnWasmExecChange(t *testing.T) {
	c := client.New(nil)

	// Inject FakeCompilers
	fakeLarge := newFakeCompiler()
	fakeMedium := newFakeCompiler()
	fakeSmall := newFakeCompiler()
	c.SetBuilders(fakeLarge, fakeMedium, fakeSmall)
	c.SetActiveBuilder(fakeLarge)

	var wasmExecChangeCalled int
	c.OnWasmExecChange = func() {
		wasmExecChangeCalled++
	}

	var onCompileErr error
	var onCompileCalled int
	c.OnCompile = func(err error) {
		onCompileCalled++
		onCompileErr = err
	}

	// 1. Compile error via NewFileEvent. Assert OnWasmExecChange NOT called, OnCompile called with error.
	someErr := errors.New("compile error")
	fakeLarge.CompileErr = someErr
	wasmExecChangeCalled = 0
	onCompileCalled = 0

	err := c.NewFileEvent("main.go", ".go", "main.go", "write")
	if err == nil {
		t.Error("Expected error from NewFileEvent when compile fails, got nil")
	}
	if wasmExecChangeCalled != 0 {
		t.Errorf("OnWasmExecChange called %d times, expected 0 on compile error", wasmExecChangeCalled)
	}
	if onCompileCalled != 1 {
		t.Errorf("OnCompile called %d times, expected 1", onCompileCalled)
	}
	if onCompileErr != someErr {
		t.Errorf("OnCompile err = %v, expected %v", onCompileErr, someErr)
	}

	// 2. Compile success via NewFileEvent, but no mode change. Assert OnWasmExecChange NOT called.
	fakeLarge.CompileErr = nil
	wasmExecChangeCalled = 0
	onCompileCalled = 0

	err = c.NewFileEvent("main.go", ".go", "main.go", "write")
	if err != nil {
		t.Errorf("Unexpected error from NewFileEvent: %v", err)
	}
	if wasmExecChangeCalled != 0 {
		t.Errorf("OnWasmExecChange called %d times, expected 0 on successful compile without mode change", wasmExecChangeCalled)
	}
	if onCompileCalled != 1 {
		t.Errorf("OnCompile called %d times, expected 1", onCompileCalled)
	}
	if onCompileErr != nil {
		t.Errorf("OnCompile err = %v, expected nil", onCompileErr)
	}

	// 3. Force mode switch via Change with success. Assert OnWasmExecChange IS called.
	fakeMedium.CompileErr = nil
	wasmExecChangeCalled = 0
	onCompileCalled = 0

	c.Change("M")

	if wasmExecChangeCalled != 1 {
		t.Errorf("OnWasmExecChange called %d times, expected 1 on mode change success", wasmExecChangeCalled)
	}
	if onCompileCalled != 1 {
		t.Errorf("OnCompile called %d times, expected 1", onCompileCalled)
	}
	if onCompileErr != nil {
		t.Errorf("OnCompile err = %v, expected nil", onCompileErr)
	}

	// 4. Force mode switch via Change with error. Assert OnWasmExecChange NOT called.
	fakeSmall.CompileErr = someErr
	wasmExecChangeCalled = 0
	onCompileCalled = 0

	c.Change("S")

	if wasmExecChangeCalled != 0 {
		t.Errorf("OnWasmExecChange called %d times, expected 0 on mode change failure", wasmExecChangeCalled)
	}
	if onCompileCalled != 1 {
		t.Errorf("OnCompile called %d times, expected 1", onCompileCalled)
	}
	if onCompileErr != someErr {
		t.Errorf("OnCompile err = %v, expected %v", onCompileErr, someErr)
	}

	// 5. Change to SAME mode. Assert OnWasmExecChange NOT called.
	fakeMedium.CompileErr = nil
	wasmExecChangeCalled = 0
	onCompileCalled = 0
	// current mode is "M" (or "S" from previous test, let's make sure)
	c.SetMode("M")
	wasmExecChangeCalled = 0

	c.Change("M")
	if wasmExecChangeCalled != 0 {
		t.Errorf("OnWasmExecChange called %d times, expected 0 when changing to the same mode", wasmExecChangeCalled)
	}
	if onCompileCalled != 1 {
		t.Errorf("OnCompile called %d times, expected 1", onCompileCalled)
	}
}
