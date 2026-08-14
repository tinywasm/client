---
PLAN: "refactor!: disolver client — la compilación ya vive en gobuild"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 8257303978851486050
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
>
> Forma parte de un cambio con ruptura coordinado desde
> https://github.com/tinywasm/core/blob/main/docs/PLAN.md

# Plan — disolver `tinywasm/client`

## Por qué no se crea un `clientc`

La tentación era crear un repo nuevo "con solo la lógica para compilar a TinyGo".
**Sería un tercer repo haciendo lo que ya hacen dos.**

```go
// tinywasm/gobuild — ESTO es el compilador
func New(c *Config) *GoBuild
func (h *GoBuild) CompileProgram() error
func (h *GoBuild) BuildArguments() []string
func (h *GoBuild) Cancel() error

// tinywasm/tinygo — ESTO es el toolchain
func IsInstalled() bool ; func GetPath() ; func Install()
```

Y este repo **no compila nada**: configura tres `gobuild.New(...)` y delega.

```go
// builderInit.go
w.builderSizeLarge  = gobuild.New(&codingConfig)   // Go estándar
w.builderSizeMedium = gobuild.New(&debugConfig)    // TinyGo debug
w.builderSizeSmall  = gobuild.New(&prodConfig)     // TinyGo prod
```

Un `clientc` no tendría una razón para cambiar propia: cambiaría cuando cambie
TinyGo (→ `tinygo`), cuando cambien los flags (→ `gobuild`), o cuando cambie el
artefacto deseado (→ `sitec`). Tres disparadores, todos ajenos. Un repo se
justifica por una razón propia, no por agrupar llamadas a otros.

## Qué es este repo en realidad

1.605 líneas, y la compilación no está entre ellas:

| Archivo | Líneas | Qué es realmente |
|---|---|---|
| `client.go`, `builderInit.go`, `wasmbuild.go`, `client_extensions.go` | 647 | selección de modo y orquestación sobre `gobuild` |
| `Change.go` | 182 | control de TUI (`Label`, `Options`, `Change`, `Shortcuts`) |
| `tiny_verify_proyect.go` | 159 | verificar que el proyecto es compatible con TinyGo |
| `generator.go`, `web_client_generator.go` | 139 | scaffolding del cliente |
| `storage.go` | 100 | memoria vs disco |
| `vscode_config.go` | 84 | **escribe la configuración de VSCode** |
| `file_event.go` | 78 | eventos de archivo (interfaz de `devwatch`) |
| `http.go` | 47 | servir el binario |
| `mcp-tool.go`, `models*.go`, `javascripts.go`, `config.go` | ~130 | MCP, DTOs, flags |

Un compilador que escribe la config de VSCode e importa `markdown`, `mcp`,
`router` y `tui` no es un compilador. Es el mismo diagnóstico que tenía
`tinywasm/assetmin`, ya archivado por la misma razón.

## El código YA FUE MOVIDO

**No muevas nada — ya está movido.** Este plan consiste en **borrar** de aquí lo
que ya vive en otro repo.

| Destino | Qué se llevó |
|---|---|
| `tinywasm/app`, carpeta `from_client/` | 16 archivos de código + 22 tests: TUI, VSCode, MCP, HTTP, eventos, storage, generador, y la orquestación de modos |
| `tinywasm/sitec`, `select_tinygo_verify.go` | `tiny_verify_proyect.go` — verificación de compatibilidad |
| `tinywasm/gobuild` | **nada**: ya tenía la compilación |

Ya aplicado en el movimiento: cláusula de paquete a `app` / `app_test` / `sitec`.

## Etapas

### Etapa 1 — borrar el código movido

```
client.go  builderInit.go  wasmbuild.go  client_extensions.go
Change.go  vscode_config.go  mcp-tool.go  http.go  file_event.go
storage.go  generator.go  web_client_generator.go  javascripts.go
models.go  models_orm.go  config.go  tiny_verify_proyect.go
tests/
```

Verifica antes de borrar que los destinos existen:

```sh
ls ../app/from_client/*.go        # 16 archivos
ls ../app/from_client/tests/*.go  # 22 tests
ls ../sitec/select_tinygo_verify.go
```

### Etapa 2 — borrar `cmd/wasmbuild` y su lógica

**Decisión tomada: se borra.** No queremos redundancia de herramientas; para eso
la compilación por línea de comando se traslada a `sitec build`.

Se borra el CLI **y la función que solo existía para servirlo**:

```
cmd/wasmbuild/          (main.go + README.md)
wasmbuild.go            (RunWasmBuild, RunWasmBuildHooks, RunWasmBuildClient,
                         WasmBuildArgs, SetRunWasmBuildHooks)
javascripts.go          (ParseWasmSizeModeFlag — flag del CLI)
```

Verificado: `RunWasmBuild` solo lo llaman `cmd/wasmbuild/main.go` y sus tests.
Nada del arnés lo usa.

#### Precondición — `sitec build` debe cubrir los cinco pasos

`RunWasmBuild` hace más que compilar. **No borres nada hasta que `sitec build`
haga los cinco**, o se pierde capacidad en silencio:

| # | Paso | Detalle |
|---|---|---|
| 1 | `EnsureTinyGoInstalled()` | instala TinyGo si falta — crítico en CI, donde no está |
| 2 | verificar `web/client.go` | fallar con un mensaje claro si no existe |
| 3 | crear `web/public` | el directorio de salida |
| 4 | **generar `script.js`** | `js.SetRuntime(runtimeFromMode(mode))` + `js.PageBootstrap().Content` |
| 5 | compilar el WASM | con `TINYGOROOT` y `PATH` inyectados en el entorno |

El paso 4 es el que más fácil se pierde: **el runtime JS depende del modo de
compilación**. TinyGo y Go estándar emiten `wasm_exec` distintos, así que un
binario con el loader equivocado no arranca. Por eso el puerto `WasmBuilder` de
`sitec` devuelve el binario **y** su glue, no solo el binario.

Su test se movió a `sitec/tests/build_wasm_test.go`: ahí es el criterio de
aceptación de `sitec build`, incluido `TestRunWasmBuild_IncludesTinyGoEnv`.

#### Referencias a limpiar

`wasmbuild` aparece en `README.md`, `cmd/wasmbuild/README.md` y
`docs/stages/stage1_replace_calls.md` / `stage2_cleanup.md`. Ninguna fuera de
este repo: no hay instalador, CI ni otro módulo que lo invoque, así que el
borrado no rompe consumidores.

**Aceptación:** `grep -rn "wasmbuild" .` devuelve vacío en todo el repo.

### Etapa 3 — `README.md` y archivar

`README.md` pasa a ser un aviso corto: el repo fue disuelto, la compilación vive
en `github.com/tinywasm/gobuild`, el arnés en `github.com/tinywasm/app`, y la
verificación de compatibilidad en `github.com/tinywasm/sitec`.

Archivar en GitHub **solo cuando** ningún `go.mod` del ecosistema lo referencie:

```sh
grep -rn "tinywasm/client" ~/Dev/Project/tinywasm/*/go.mod
```

Hoy lo referencian `app`, `deploy`, `website` y `goflare`. `app` lo suelta en su
propia etapa; los otros tres deben verificarse: si solo es dependencia indirecta,
se resuelve con `go mod tidy`.

## No hacer

- **No crees `tinywasm/clientc`** ni ningún repo intermedio entre `sitec` y
  `gobuild`. La justificación está arriba y es el motivo de este plan.
- **No dejes un paquete `client` que reenvíe** a los nuevos destinos. Es un
  cambio con ruptura declarado; un puente mantiene vivas las dos rutas.
- No añadir llamadas a `gopush` ni a `codejob`.

## Etapas

| # | Alcance | Aceptación |
|---|---|---|
| 1 | Borrar el código ya movido | `ls *.go` solo deja lo de la etapa 2 |
| 2 | Borrar `cmd/wasmbuild`, `wasmbuild.go` y `javascripts.go` | `grep -rn "wasmbuild" .` vacío; `sitec build` cubre los 5 pasos |
| 3 | README de disolución y archivar | ningún `go.mod` referencia `tinywasm/client` |
