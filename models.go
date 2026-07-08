package client

// SetModeArgs defines arguments for the wasm_set_mode tool.
// The JSON input schema is derived from the generated Schema() (models_orm.go)
// by tinywasm/mcp — no hand-written schema strings.
type SetModeArgs struct {
	Mode string `input:"required,enum=L;M;S"`
}
