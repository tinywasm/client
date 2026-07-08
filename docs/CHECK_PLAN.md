# PLAN — Migrate `WasmClient` compiler-mode field to `HandlerSelection` + `SetModeArgs` to typed `model.Definition`

> This plan is dispatched via the CodeJob workflow. See skill: `agents-workflow`.
> Part of a multi-repo effort tracked by the maintainer outside this repo — you do NOT
> need (and will not have) that tracking document; everything required is inline here.
> **Depends on the `devtui` GATE phase** (`HandlerSelection` must exist and be published first).
> Section 9 (model migration) has no gate and must be done regardless.

You are an external agent with **zero prior context**. Everything you need is here.

---

## 1. Goal

`WasmClient` exposes a compiler-mode setting with three mutually-exclusive modes:
`L` (Large / stdlib), `M` (Medium / TinyGo), `S` (Small / TinyGo). Today it is a
**text-edit** field (`HandlerEdit`): the DevTUI footer shows a single letter and the user
must type `L`/`M`/`S` to change it — wasteful and unintuitive.

Migrate it to a **radio / segmented control** by implementing the new `HandlerSelection`
interface, so DevTUI renders all three modes as labelled buttons and the user picks one with
`Enter` → `←/→` → `Enter`. The existing global letter shortcuts (`L`/`M`/`S`) stay.

---

## 2. Critical constraint — DO NOT import `devtui`

`WasmClient` satisfies DevTUI interfaces **structurally (duck typing)** and MUST NOT import
`github.com/tinywasm/devtui`. Verify with `grep -rn devtui *.go` → must return nothing.

`HandlerSelection` was designed specifically so this works with **only stdlib types**. Its
method set (see the shape below) references no `devtui` types.

The `HandlerSelection` contract (defined in `devtui/interfaces.go`) is:

```go
type HandlerSelection interface {
	Name() string                 // "CLIENT"
	Label() string                // "Compiler Mode"
	Value() string                // active option key, e.g. "L"
	Change(newValue string)       // called on confirm with the selected key
	Options() []map[string]string // ordered {value: label}
}
```

`WasmClient` already implements `Name()`, `Label()`, `Value()`, and `Change(newValue)`
(see files below). **The ONLY new method required is `Options()`.**

---

## 3. Current state (verified — reuse this, do not rewrite)

**`client.go`** — the field methods already exist:

```go
func (w *WasmClient) Name() string  { return "CLIENT" }
func (w *WasmClient) Label() string { return "Compiler Mode" }

func (w *WasmClient) Value() string {
	w.storageMu.RLock()
	defer w.storageMu.RUnlock()
	if w.CurrentSizeMode == "" {
		return w.buildLargeSizeShortcut // "L"
	}
	return w.CurrentSizeMode
}
```

Mode identifiers are fields on `WasmClient` (constructed in `client.go`):
`buildLargeSizeShortcut = "L"`, `buildMediumSizeShortcut = "M"`, `buildSmallSizeShortcut = "S"`.

**`Change.go`** — already implements both the change logic and the shortcut provider:

```go
func (w *WasmClient) Shortcuts() []map[string]string {
	return []map[string]string{
		{w.buildLargeSizeShortcut: lang.Translate("mode", "Large", "stLib").String()},
		{w.buildMediumSizeShortcut: lang.Translate("mode", "Medium", "tinygo").String()},
		{w.buildSmallSizeShortcut: lang.Translate("mode", "Small", "tinygo").String()},
	}
}

// Change updates the compiler mode (validates, recompiles, notifies). Unchanged.
func (w *WasmClient) Change(newValue string) { /* ... existing logic ... */ }
```

Note: `Shortcuts()` already returns the **exact `[]map[string]string` shape** that
`Options()` needs — `{value: label}` per mode, in order. Reuse it.

---

## 4. The change

### 4.1 Add `Options()` in `Change.go`

Place it right after `Shortcuts()`:

```go
// Options returns the compiler-mode choices as ordered {value: label} pairs so
// DevTUI renders them as a radio / segmented control (HandlerSelection). The
// data is identical to Shortcuts(): each mode's key is the value passed to
// Change() and its translated caption is the button label.
func (w *WasmClient) Options() []map[string]string {
	return w.Shortcuts()
}
```

That is the whole functional change. Because `WasmClient` now also implements
`Options()`, DevTUI's `AddHandler` detects it as `HandlerSelection` (its type switch checks
`HandlerSelection` before `HandlerEdit`) and renders buttons instead of a text input.

### 4.2 Update the doc comment on `Change`

`Change.go` line ~17 currently says:

```go
// Change updates the compiler mode for WasmClient.
// Implements the HandlerEdit interface: Change(newValue string)
```

Change the second line to reflect the new role:

```go
// Change updates the compiler mode for WasmClient.
// Implements HandlerSelection.Change: called with the selected option's value
// ("L"/"M"/"S") when the user confirms a mode (radio) or presses a global shortcut.
```

### 4.3 Nothing else changes

- `Value()`, `Label()`, `Name()` are already correct and reused as-is.
- Keep `Shortcuts()` — global `L`/`M`/`S` keys must continue to work (decision: keep both).
- `ValidateMode`, `UpdateCurrentBuilder`, storage, recompilation logic — untouched.

---

## 5. Behavior after migration (for verification)

- The compiler-mode field renders three buttons in the footer: `Large` · `Medium` · `Small`,
  with the active mode highlighted.
- `Enter` on the field enters selection mode; `←/→` move the highlight; `Enter` confirms and
  calls `Change(selectedValue)` (recompiles, notifies listeners); `Esc` cancels with no change.
- Pressing `L`/`M`/`S` from any tab still switches mode (global shortcut → same `Change`).
- The active button always reflects `Value()` (`CurrentSizeMode`) after a change.

---

## 6. Tests

Add/extend a test in the `client` package (see existing `*_test.go` for style):

1. **`Options()` shape**: returns 3 ordered single-entry maps with keys `L`, `M`, `S` and the
   translated labels; assert it equals `Shortcuts()`.
2. **Value/Options consistency**: after `Change("M")`, `Value()` returns `"M"` and the key
   `"M"` is present in `Options()`.
3. Existing mode tests (`client_mode_test.go`, `changefunc_control_test.go` if present) must
   still pass — the `Change` logic is unchanged.

Run `gotest ./...` (never bare `go test`; install:
`go install github.com/tinywasm/devflow/cmd/gotest@latest`). All green.

> There is no DevTUI-rendering test here (client must not import devtui). The rendering and
> keyboard behavior are covered by `selection_handler_test.go` in the `devtui` repo.

---

## 7. Code-quality rules (enforced)

- **Must NOT import `devtui`** — verify with `grep -rn devtui *.go` (empty result).
- **No new hardcoded strings**: reuse `w.buildLargeSizeShortcut` / `Medium` / `Small` and the
  existing `lang.Translate(...)` labels via `Shortcuts()`. Do not duplicate the `{value:label}`
  literals in `Options()` — delegate to `Shortcuts()`.
- **No standard library**: this package uses `github.com/tinywasm/fmt` and
  `github.com/tinywasm/fmt/lang`; do not add stdlib `fmt`/`strings`/`errors`.
- **Backward compatible**: `Change`, `Value`, `Label`, `Name`, `Shortcuts` signatures unchanged.
- **Models are typed Definitions**: no struct tags (`input:"..."`) as source of truth, no
  hand-written `Schema()`/`Validate()`/schema strings — `model.Definition` + ormc only
  (Section 9).

---

## 9. Model migration — `SetModeArgs` becomes a typed `model.Definition`

**Mandatory ecosystem pattern** (reference, web:
<https://github.com/tinywasm/model/blob/main/README.md> and
<https://github.com/tinywasm/orm/blob/main/docs/ARQUITECTURE.md> — the rule is restated
fully here, reading them is optional): the source of truth for a model is a **typed
`model.Definition` literal** (var name MUST end in `Model` — that is how `ormc` discovers
it). `ormc` generates the concrete struct plus `Schema()`, `Pointers()`, `Validate()`,
`EncodeFields()`/`DecodeFields()`. Hand-written structs with `input:"..."` string tags are
the OLD, removed pattern.

`models.go` today is that old pattern (hand-written struct + `input:"required,enum=L;M;S"`
tag). Note the enum constraint is already **silently lost** in the generated schema
(`models_orm.go` has only `NotNull`) — the typed Definition restores it.

### 9.1 Replace `models.go` content

```go
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
			Type:    model.FieldText,
			NotNull: true,
			Permitted: model.Permitted{
				Extra:   []rune{'L', 'M', 'S'},
				Minimum: 1,
				Maximum: 1,
			},
		},
	},
}
```

- **Delete the hand-written `SetModeArgs` struct** — ormc generates it (names would
  collide otherwise).
- No `Widget` (this is a tool-args model, not a form) and no `DB` (not a table).
- Regenerate: `go install github.com/tinywasm/orm/cmd/ormc@latest && ormc` (from the
  module root). Never edit `models_orm.go` by hand.
- `mcp-tool.go` needs **no change**: `Args: new(SetModeArgs)` keeps compiling against the
  generated struct (field `Mode string` round-trips).

### 9.2 Tests for the restored validation

In the existing test style (`gotest ./...`, stdlib-free package rules apply):

1. `model.ValidateFields('u', &SetModeArgs{Mode: "L"})` → nil; `Mode: "X"` and `Mode: "LM"`
   → error (the enum lives again, typed).
2. The `wasm_set_mode` tool binds args via `req.Bind`: a request with `{"mode":"X"}` returns
   an error, `{"mode":"M"}` reaches `w.Change` (extend the existing mcp tool test if present;
   otherwise add a focused one).

---

## 10. Execution stages

| Stage | File | Deliverable |
|---|---|---|
| 1 | `Change.go` | Add `Options()` delegating to `Shortcuts()` (Section 4.1). |
| 2 | `Change.go` | Update the `Change` doc comment (Section 4.2). |
| 3 | `*_test.go` | Tests for `Options()` shape + Value/Options consistency (Section 6). |
| 4 | `models.go` (Definition), `models_orm.go` (regenerated by ormc) | `SetModeArgsModel` typed Definition; hand-written struct deleted (Section 9). |
| 5 | `*_test.go` | Mode-validation tests via generated `Validate` + `req.Bind` (Section 9.2); `gotest ./...` green. |

Do NOT run `gopush` or `codejob` — those are local developer tools managed outside this task.
