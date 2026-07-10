package client

import "github.com/tinywasm/model"

// SetModeArgsModel defines the arguments of the wasm_set_mode MCP tool.
// ormc generates the SetModeArgs struct + Schema/Pointers/Validate/codec from
// this literal; the MCP inputSchema is derived from the generated Schema().
// mode is exactly one of L/M/S — expressed as: length exactly 1, allowed
// characters only 'L','M','S' (Permitted has no enum concept; this is the
// faithful typed equivalent for single-letter modes).
var SetModeArgsModel = model.Definition{
	Name: "set_mode_args",
	Fields: model.Fields{
		{
			Name:    "mode",
			Type:    model.Text(),
			NotNull: true,
			Permitted: model.Permitted{
				Extra:   []rune{'L', 'M', 'S'},
				Minimum: 1,
				Maximum: 1,
			},
		},
	},
}
