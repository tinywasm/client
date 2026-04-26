# PLAN: client — Exponer OnCompile callback para observabilidad de compilación

## Problema

No existe forma programática de saber cuándo `WasmClient` termina una
compilación disparada por un evento de archivo. El único observable actual es
`LogSuccessState`, cuyo formato es un detalle de implementación interno.

## Diseño: `OnCompile func(error)`

`WasmClient` ya usa el patrón callback para eventos similares:
- `Config.Callback func(error)` — callback del builder `gobuild` (async)
- `Config.OnWasmExecChange func()` — notifica cambio de `wasm_exec.js`

Añadir `OnCompile func(error)` en `WasmClient` (no en `Config`) que se
invoque al terminar cada compilación iniciada por `NewFileEvent`, sea éxito
o error.

```
NewFileEvent
    → s.Compile()           (éxito o error)
    → w.OnCompile(err)      ← hook: nil=ok, err=falló
    → w.LogSuccessState()   (solo si err==nil)
```

El hook se llama **siempre** (éxito y error) para permitir contar tanto
intentos exitosos como fallidos.

## Cambios requeridos

### `client/client.go` — campo `OnCompile`

```go
type WasmClient struct {
    ...
    // OnCompile se invoca al terminar cada compilación disparada por NewFileEvent.
    // err==nil indica éxito; err!=nil indica fallo.
    OnCompile func(err error)
    ...
}
```

### `client/file_event.go` — llamar el hook

```go
func (w *WasmClient) NewFileEvent(...) error {
    ...
    compileErr := s.Compile()

    if w.OnCompile != nil {
        w.OnCompile(compileErr)
    }

    if compileErr != nil {
        return Err("compiling to WebAssembly error: ", compileErr)
    }

    w.LogSuccessState()
    ...
}
```

### `client/client_extensions.go` — setter público

```go
// SetOnCompile registra un callback invocado al terminar cada compilación
// disparada por un evento de archivo. err==nil indica éxito.
func (w *WasmClient) SetOnCompile(fn func(err error)) {
    w.OnCompile = fn
}
```

## Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `client/client.go` | Campo `OnCompile func(err error)` en `WasmClient` |
| `client/client_extensions.go` | `SetOnCompile(fn func(error))` |
| `client/file_event.go` | Llamar `w.OnCompile(compileErr)` antes del return |

## Orden de ejecución

| # | Tarea | Archivo | Estado |
|---|-------|---------|--------|
| 1 | Campo `OnCompile func(err error)` en `WasmClient` | `client/client.go` | Pendiente |
| 2 | `SetOnCompile(fn)` | `client/client_extensions.go` | Pendiente |
| 3 | `w.OnCompile(compileErr)` en `NewFileEvent` | `client/file_event.go` | Pendiente |
| 4 | Publicar nueva versión (`gopush`) | — | Pendiente |
