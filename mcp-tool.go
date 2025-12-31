package client

// ToolExecutor defines how a tool should be executed
type ToolExecutor func(args map[string]any)

// ToolMetadata provides MCP tool configuration metadata
// This is the standard interface that mcpserve expects
type ToolMetadata struct {
	Name        string
	Description string
	Parameters  []ParameterMetadata
	Execute     ToolExecutor // Execution function
}

// ParameterMetadata describes a tool parameter
type ParameterMetadata struct {
	Name        string
	Description string
	Required    bool
	Type        string
	EnumValues  []string
	Default     any
}

// GetMCPToolsMetadata returns metadata for all WasmClient MCP tools
func (w *WasmClient) GetMCPToolsMetadata() []ToolMetadata {
	return []ToolMetadata{
		{
			Name: "wasm_set_mode",
			Description: "Change WebAssembly compilation mode for the Go frontend. " +
				"L=LARGE (Go std, ~2MB, full features), " +
				"M=MEDIUM (TinyGo debug, ~500KB, most features), " +
				"S=SMALL (TinyGo compact, ~200KB, minimal). " +
				"Use single letter shortcuts: L, M, or S.",
			Parameters: []ParameterMetadata{
				{
					Name:        "mode",
					Description: "Compilation mode: L (large), M (medium), or S (small)",
					Required:    true,
					Type:        "string",
					EnumValues:  []string{"L", "M", "S"},
				},
			},
			Execute: func(args map[string]any) {
				modeValue, ok := args["mode"]
				if !ok {
					w.Logger("missing required parameter 'mode'. Use L, M, or S")
					return
				}

				mode, ok := modeValue.(string)
				if !ok {
					w.Logger("parameter 'mode' must be a string (L, M, or S)")
					return
				}

				// Domain-specific logic: Change WASM compilation mode
				// Messages flow through w.Logger() which is captured by mcpserve
				w.Change(mode)
			},
		},
	}
}
