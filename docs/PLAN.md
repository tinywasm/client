# PLAN — `tinywasm/client` se reduce a build-only

## Objetivo

Recortar `tinywasm/client` a una sola responsabilidad: **compilar el binario
WASM** según el modo seleccionado (L = Go stdlib, M/S = TinyGo) y servirlo
en `/client.wasm`. Toda composición de JavaScript se elimina y se delega en
`tinywasm/js`, que es desde ahora la única API JS del framework.

## Justificación

Hoy `client` mezcla dos responsabilidades:
1. Build de WASM (compilación, watcher, modo Go/TinyGo, servir
   `/client.wasm`).
2. Composición JS (`Javascript` struct, `GetSSRClientInitJS()`, embeds de
   `wasm_exec_*.js`, escritura de `wasm_exec.js` a disco).

La segunda es responsabilidad natural de `tinywasm/js` (espejo de
`tinywasm/css` que posee toda la composición CSS). Mantenerla en `client`:

- Duplica responsabilidades.
- Obliga a `client` a embeber `wasm_exec.js` — generando dos fuentes de
  verdad cuando `js` también lo necesita para componer shims de SW/Worker.
- Fuerza a `app` a cablear un callback opaco (`GetSSRClientInitJS`) entre
  `client` y `assetmin` — coordinación cross-package innecesaria.

Tras este PLAN, `client` queda borrable un día sin tocar JS. `js` queda
borrable un día sin tocar build. Acoplamiento por dependencia explícita
(`client` requires `js` para `wasmbuild.go`), sin estado compartido.

## Precondiciones (publicado antes de empezar)

`tinywasm/js v0.2.0` con esta superficie pública:

```go
package js

type Script struct { Name, Content string }
func (s *Script) String() string

type Runtime int
const (RuntimeGo Runtime = iota; RuntimeTinyGo)
func SetRuntime(r Runtime)              // write-once-at-boot

func PageBootstrap() *Script             // Name="" → bundle página
func ServiceWorker(h ServiceWorkerHandler) *Script
func WebWorker(name string, h WebWorkerHandler) *Script

// ... ServiceWorkerHandler / WebWorkerHandler / Request / Response / Message
```

`PageBootstrap()` devuelve `*Script{Name:"", Content: <wasm_exec> + <bootstrap>}`
listo para escribir o registrar con assetmin. Verifica:
`go doc github.com/tinywasm/js`.

## Eliminaciones (sin reexports, sin aliases — full breaking)

### Archivos a borrar

- `client/assets/wasm_exec_go.js` (ahora vive en `js/assets/`).
- `client/assets/wasm_exec_tinygo.js` (idem).

### Símbolos a eliminar en `client/javascripts.go`

| Símbolo | Razón |
|---|---|
| `//go:embed assets/wasm_exec_go.js` + `var embeddedWasmExecGo []byte` | Embeds movidos a `js`. |
| `//go:embed assets/wasm_exec_tinygo.js` + `var embeddedWasmExecTinyGo []byte` | Idem. |
| `func WasmExecGoSignatures() []string` | Lista interna; consumidores externos que la necesitan declaran inline las firmas que les interesan. |
| `func WasmExecTinyGoSignatures() []string` | Idem. |
| `type Javascript struct` | Composición JS sale de client. |
| `func (*Javascript) SetMode` | Idem. |
| `func (*Javascript) SetWasmFilename` | Idem. |
| `func NewJavascriptFromArgs` | Idem. |
| `func (*Javascript) ArgumentsForServer` | El equivalente sigue siendo necesario para el flag `-wasmsize_mode`, pero se traslada a `WasmClient` directamente (no requiere struct intermedia). |
| `func (*Javascript) getWasmExecContent` | Lectura de embeds ya no aplica. |
| `func (*WasmClient) getWasmExecContent` | Idem. |
| `func (*Javascript) GetSSRClientInitJS` | Composición JS reemplazada por `js.PageBootstrap()`. |
| `func (*WasmClient) GetSSRClientInitJS` | Idem. Wrapper público también eliminado. |
| `func (*WasmClient) WasmExecJsOutputPath` | Salida de wasm_exec.js a disco ya no aplica (assetmin bundlea `js.PageBootstrap()` en `/script.js`). |
| `func (*WasmClient) wasmProjectWriteOrReplaceWasmExecJsOutput` | Idem. |
| `func (*WasmClient) ClearJavaScriptCache` | El caching JS se elimina junto con `Javascript`. |
| Campos `mode_large_go_wasm_exec_cache`, `mode_medium_tinygo_wasm_exec_cache`, `mode_small_tinygo_wasm_exec_cache` | Caches obsoletos. |

### Llamadas internas a actualizar

| Archivo | Llamada actual | Reemplazo |
|---|---|---|
| `client/wasmbuild.go:107-117` | `Javascript{}` + `SetMode(mode)` + `SetWasmFilename("client.wasm")` + `GetSSRClientInitJS()` para escribir `script.js` | `js.SetRuntime(runtimeFromMode(mode))` + `os.WriteFile(scriptPath, []byte(js.PageBootstrap().Content), 0644)` |
| `client/generator.go:82,87` | `t.wasmProjectWriteOrReplaceWasmExecJsOutput()` | Eliminar la llamada (assetmin produce el bundle ahora; `client` no escribe JS). |
| `client/Change.go:56` | Idem | Eliminar. |
| `client/client.go:109-110` | `w.Javascript = &Javascript{}; w.Javascript.SetWasmFilename(...)` | Eliminar inicialización; `Javascript` ya no existe. |
| `client/client.go:235` | `w.wasmProjectWriteOrReplaceWasmExecJsOutput()` | Eliminar. |
| `client/client.go:301-302` | `w.Javascript.SetMode(...); return w.Javascript.ArgumentsForServer()` | Mover lógica de `ArgumentsForServer` directamente a `WasmClient`. |

Helper local sugerido en `client/wasmbuild.go`:
```go
func runtimeFromMode(mode string) js.Runtime {
    if mode == "L" { return js.RuntimeGo }
    return js.RuntimeTinyGo  // "M" y "S"
}
```

## Lo que `client` SÍ conserva

- Detección Go vs TinyGo (campo `TinyGoCompilerFlag`, `Value()`, modos L/M/S).
- Compilación WASM (`Compile`, `wasmbuild.go`, `cmd/`, generación de
  `client.wasm`).
- Watcher de cambios en `web/main.wasm.go` y `wasm_exec.js` (si aplica;
  evaluar si el watcher de `wasm_exec.js` aún tiene sentido — el archivo
  vive en `js/assets/`, ya no en disco bajo control del usuario).
- Hook `OnWasmExecChange` (lo dispara `app` para resincronizar runtime
  global vía `js.SetRuntime`).
- `RegisterRoutes(mux)` que sirve `/client.wasm`.
- Storage en memoria vs disco (`UseDiskStorage`, etc.).
- Templates de proyecto (`templates/`, `web_client_generator.go`,
  `builderInit.go`) — generan el `main.wasm.go` del usuario, no JS.

## Cambios en consumidores rotos por el breaking

Búsqueda obligatoria al inicio del PR:
```bash
grep -rn "embeddedWasmExecGo\|embeddedWasmExecTinyGo\|WasmExecGoSignatures\|WasmExecTinyGoSignatures\|GetSSRClientInitJS\|\.Javascript\b\|WasmExecJsOutputPath" client/
```

Hits conocidos en tests al redactar este PLAN:
- `tests/debug_test.go:82`, `tests/initialization_test.go:64`,
  `tests/javascripts_test.go:64-93`, `tests/wasmbuild_test.go:104,158`.

Acciones por test:
- Si testea producción de `script.js` (composición JS) → mover a `js/tests/`
  contra `js.PageBootstrap()`, o borrar (la spec ya vive en
  `js/tests/wasm_exec_test.go`).
- Si testea build/compilación → permanecer en `client/tests/` y eliminar
  cualquier referencia a símbolos eliminados; declarar inline las 2-3
  firmas de runtime que necesite para verificar el WASM compilado.

## Tests nuevos

| Archivo | Test | Verifica |
|---|---|---|
| `tests/migration_test.go` | `TestNoLocalEmbeds` | `grep` programático: el package `client` no contiene `//go:embed assets/wasm_exec_*.js`. |
| `tests/migration_test.go` | `TestNoJavascriptStruct` | El package `client` no exporta `Javascript`, `GetSSRClientInitJS`, `WasmExec*Signatures`. |
| `tests/wasmbuild_test.go` | `TestWasmbuild_WritesScriptJSFromJSPackage` | Tras `wasmbuild` en modo L, `script.js` en outputDir contiene runtime Go (firmas inline en el test); en modo S contiene runtime TinyGo. |

Ejecución: `gotest ./...` (skill `testing`).

## Reglas de dependencias

`tinywasm/client` compila a host (no WASM frontend) — la stdlib **no** está
vetada aquí. Pero las nuevas llamadas a `js` se hacen vía
`github.com/tinywasm/js`. Añadir al `client/go.mod`:

```
require github.com/tinywasm/js vX.Y.Z
```

## Stages

| # | Tarea | Done |
|---|---|---|
| 1 | Verificar `tinywasm/js v0.2.0` publicado con `PageBootstrap`, `SetRuntime`, `Runtime` constants | [x] |
| 2 | Añadir `require github.com/tinywasm/js` en `client/go.mod` | [x] |
| 3 | Eliminar archivos `client/assets/wasm_exec_*.js` | [x] |
| 4 | Eliminar todos los símbolos listados en §"Símbolos a eliminar" | [x] |
| 5 | Actualizar las 6 llamadas internas listadas en §"Llamadas internas a actualizar" (incluye `wasmbuild.go` usando `js.PageBootstrap()`) | [x] |
| 6 | Migrar `ArgumentsForServer` desde `Javascript` a `WasmClient` directamente | [x] |
| 7 | Borrar tests obsoletos en `client/tests/` o migrarlos a `js/tests/` según §"Cambios en consumidores rotos" | [x] |
| 8 | Crear los 3 tests nuevos descritos en §"Tests nuevos" | [x] |
| 9 | `gotest ./...` verde | [x] |
| 10 | Actualizar `README.md` documentando que `client` es build-only | [x] |
| 11 | Publicar nueva versión con `gopush` | [ ] |
