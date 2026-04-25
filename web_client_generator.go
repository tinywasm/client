package client

type webClientGenerator struct {
	client *WasmClient
}

// Name returns same value as WasmClient.Name() for HeadlessTUI dispatch key matching
func (w *webClientGenerator) Name() string {
	return w.client.Name()
}

// Label returns the label for the generator action
func (w *webClientGenerator) Label() string {
	return "Generate web/client.go"
}

// Execute calls CreateDefaultWasmFileClientIfNotExist(true) to skip IDE config.
func (w *webClientGenerator) Execute() {
	w.client.CreateDefaultWasmFileClientIfNotExist(true)
}

// WebClientGenerator returns a handler that generates the default web/client.go file.
func (w *WasmClient) WebClientGenerator() *webClientGenerator {
	return &webClientGenerator{client: w}
}
