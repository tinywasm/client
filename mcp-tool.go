package client

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

// GetMCPTools returns metadata for all WasmClient MCP tools
func (w *WasmClient) GetMCPTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "wasm_set_mode",
			Description: "Change WebAssembly compilation mode for the Go frontend. " +
				"L=LARGE (Go std, ~2MB, full features), " +
				"M=MEDIUM (TinyGo debug, ~500KB, most features), " +
				"S=SMALL (TinyGo compact, ~200KB, minimal). " +
				"Use single letter shortcuts: L, M, or S.",
			InputSchema: new(SetModeArgs).InputSchema(),
			Resource:    "wasm",
			Action:      'u',
			Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
				var args SetModeArgs
				if err := req.Bind(&args); err != nil {
					return nil, err
				}

				// Domain-specific logic: Change WASM compilation mode
				// Messages flow through w.Logger() which is captured by mcpserve
				w.Change(args.Mode)
				return mcp.Text("Compilation mode changed to " + args.Mode), nil
			},
		},
	}
}
