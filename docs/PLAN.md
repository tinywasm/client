# PLAN — Migrate `WasmClient` compiler-mode field to `HandlerSelection`

> This plan is dispatched via the CodeJob workflow. See skill: `agents-workflow`.
> Part of the multi-repo effort tracked in `tinywasm/docs/SELECTION_HANDLER_MASTER_PLAN.md`.
> **Depends on the `devtui` GATE phase** (`HandlerSelection` must exist and be published first).

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

Run `gotest ./...` (or `go test ./...`). All green.

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

---

## 8. Execution stages

| Stage | File | Deliverable |
|---|---|---|
| 1 | `Change.go` | Add `Options()` delegating to `Shortcuts()` (Section 4.1). |
| 2 | `Change.go` | Update the `Change` doc comment (Section 4.2). |
| 3 | `*_test.go` | Tests for `Options()` shape + Value/Options consistency (Section 6); `go test ./...` green. |

Do NOT run `gopush` or `codejob` — those are local developer tools managed outside this task.
