---
title: Normalize LogSuccessState Output
status: proposed
date: 2026-04-25
---

# PLAN: Normalize `LogSuccessState` Output

## Problem

`LogSuccessState` appends WASM context fields (storage name, source path, binary size) to caller-provided messages as raw variadic args. Since `Logger` delegates to a user-supplied `func(...any)` — typically `fmt.Println` or similar — args are concatenated without separators, producing unreadable output:

```
http route:/client.wasmWASMIn-Memoryweb/client.go2.0 MB
```

Three additional issues compound this:

1. **Inconsistent callers** — `http.go:44` passes only `routePath`, while `storage.go:85` passes `routePath + "->" + absPath`. The trailing WASM context repeats in both cases but callers format differently.
2. **No separation between caller message and WASM context** — the appended fields run directly into the caller string.
3. **No labeled fields** — `In-Memory`, `web/client.go`, `2.0 MB` appear without keys, making it impossible to know which is which.

## Goal

Every `LogSuccessState` call produces a consistent, human-readable line:

```
[CLIENT] <event> [<mode>|<size>]
```

- `<mode>` — `mem` (MemoryStorage) or `disk` (DiskStorage)
- `<size>` — binary size from `activeSizeBuilder.BinarySize()`
- Route path and source file path are omitted from the suffix — they are already present in the caller message when relevant

Examples after the fix:

```
[CLIENT] http route: /client.wasm [mem|2.0 MB]
[CLIENT] http route: /client.wasm [disk|4.1 MB]
[CLIENT] compiled [mem|2.0 MB]
[CLIENT] Changed To Mode dev [mem|2.0 MB]
[CLIENT] Generated web/client.go [mem|2.0 MB]
```

### Why not show the file path in the suffix?

`/client.wasm` is already in the caller message. The absolute disk path (`/home/app/web/client.wasm`) adds no value for a developer or LLM — the route is what matters, and it is already shown. Keeping only `[mode|size]` makes the line scannable at a glance.

## Root Cause

```go
// Change.go:103
func (w *WasmClient) LogSuccessState(messages ...any) {
    args := append(messages, "WASM", w.Storage.Name(), w.MainInputFileRelativePath(), w.activeSizeBuilder.BinarySize())
    w.Logger(args...)   // all fields passed as raw args, no separators
}
```

`Logger` calls `w.Log(args...)` which typically uses `fmt.Sprint(args...)` — Go's default behavior joins non-string adjacent args without spaces.

## Proposed Change

### 1. Add `storageMode()` helper

```go
func (w *WasmClient) storageMode() string {
    switch w.Storage.(type) {
    case *MemoryStorage:
        return "mem"
    default:
        return "disk"
    }
}
```

### 2. Rewrite `LogSuccessState`

```go
func (w *WasmClient) LogSuccessState(messages ...any) {
    event := fmt.Sprint(messages...)
    suffix := fmt.Sprintf("[%s|%s]", w.storageMode(), w.activeSizeBuilder.BinarySize())
    w.Logger("[CLIENT]", event, suffix)
}
```

### 3. Remove `absPath` from `storage.go:85`

```go
// before
s.Client.LogSuccessState("http route:", routePath, "->", absPath)

// after
s.Client.LogSuccessState("http route:", routePath)
```

`absPath` was the only caller passing redundant path info — all others are already clean.

## Files to Change

| File | Change |
|------|--------|
| `client/Change.go` lines 103–107 | Add `storageMode()` helper + rewrite `LogSuccessState` body |
| `client/storage.go` line 85 | Remove `"->", absPath` from `LogSuccessState` call |

## Acceptance Criteria

- [ ] `LogSuccessState("http route:", "/client.wasm")` → `[CLIENT] http route: /client.wasm [mem|2.0 MB]`
- [ ] `LogSuccessState("compiled")` → `[CLIENT] compiled [mem|2.0 MB]`
- [ ] `LogSuccessState("Changed To Mode", "dev")` → `[CLIENT] Changed To Modedev [mem|2.0 MB]`
- [ ] DiskStorage produces `disk` mode label, MemoryStorage produces `mem`
- [ ] All existing tests in `client/tests/` still pass
- [ ] `fakeRunWasmBuildClient.LogSuccessState` mock signature unchanged (`...any`)

## Out of Scope

- Changing `Logger` internals or the `Log func` signature
- Adding structured logging (JSON) — this is a display normalization only
- Touching any caller outside `Change.go`
