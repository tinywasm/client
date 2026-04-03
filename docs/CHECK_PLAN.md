# PLAN: Renombrar cmd/client a cmd/wasmbuild

## Objetivo
Renombrar `cmd/client/` a `cmd/wasmbuild/` y agregar generacion de `script.js` completo (wasm_exec + inicializacion). El CLI compila `web/client.go` -> `web/public/client.wasm` y genera `web/public/script.js` listo para usar.

## Contexto y API existente

### Structs y constructores
- `Config` ([config.go:11](client/config.go#L11)) — defaults: `SourceDir()="web"`, `OutputDir()="web/public"`
- `NewConfig()` ([config.go:36](client/config.go#L36)) — crea Config con defaults
- `WasmClient` ([client.go:13](client/client.go#L13)) — struct principal, 3 builders (Large/Medium/Small)
- `New(c *Config)` ([client.go:58](client/client.go#L58)) — constructor, inicializa builders via `builderWasmInit()`
- `Javascript` ([javascripts.go:50](client/javascripts.go#L50)) — generador de JS init, campos: `useTinyGo`, `wasmFilename`, `wasmSizeMode`

### Metodos de configuracion (en WasmClient)
- `SetMode(mode string)` ([client_extensions.go:5](client/client_extensions.go#L5)) — cambia modo activo ("L"/"M"/"S"), llama `UpdateCurrentBuilder()`
- `SetBuildOnDisk(onDisk, compileNow bool)` ([client.go:180](client/client.go#L180)) — cambia a `DiskStorage`, `compileNow=false` difiere compilacion
- `SetMainInputFile(file string)` ([client.go:248](client/client.go#L248)) — default `"client.go"`
- `SetOutputName(name string)` ([client.go:254](client/client.go#L254)) — default `"client"`
- `SetLog(f func(...any))` ([client.go:131](client/client.go#L131)) — callback de logging
- `SetAppRootDir(path string)` ([client.go:242](client/client.go#L242)) — directorio raiz, reinicia builders

### Compilacion
- `Compile() error` ([client_extensions.go:21](client/client_extensions.go#L21)) — compila via `Storage.Compile()`
- `DiskStorage.Compile()` ([storage.go:63](client/storage.go#L63)) — crea directorio, llama `activeSizeBuilder.CompileProgram()`
- `builderWasmInit()` ([builderInit.go:12](client/builderInit.go#L12)) — configura 3 builders gobuild:
  - Large: `go build` con `GOOS=js GOARCH=wasm -tags dev -p 1`
  - Medium: `tinygo build -target wasm -opt=1 -p 1`
  - Small: `tinygo build -target wasm -opt=z -no-debug -panic=trap -p 1`
- `UpdateCurrentBuilder(mode)` ([builderInit.go:84](client/builderInit.go#L84)) — cambia `activeSizeBuilder` segun modo

### Generacion de JavaScript
- `Javascript.SetMode(mode)` ([javascripts.go:57](client/javascripts.go#L57)) — "M"/"S" = TinyGo, "L" = Go
- `Javascript.SetWasmFilename(name)` ([javascripts.go:64](client/javascripts.go#L64)) — nombre del .wasm en el fetch()
- `Javascript.GetSSRClientInitJS(customizations...)` ([javascripts.go:148](client/javascripts.go#L148)) — retorna JS completo: wasm_exec.js + footer con `fetch("client.wasm")` + `instantiateStreaming`
- `Javascript.getWasmExecContent()` ([javascripts.go:125](client/javascripts.go#L125)) — retorna bytes embebidos segun `useTinyGo`

### Assets embebidos
- `embeddedWasmExecGo` ([javascripts.go:17](client/javascripts.go#L17)) — `assets/wasm_exec_go.js` (Go estandar)
- `embeddedWasmExecTinyGo` ([javascripts.go:20](client/javascripts.go#L20)) — `assets/wasm_exec_tinygo.js` (TinyGo)

### TinyGo
- `EnsureTinyGoInstalled() (string, error)` ([tinygo_installer.go:24](client/tinygo_installer.go#L24)) — verifica/instala TinyGo, retorna path

### CLI existente
- `cmd/client/main.go` ([cmd/client/main.go](client/cmd/client/main.go)) — a renombrar. Ya usa: `NewConfig`, `New`, `SetMode("S")`, `SetBuildOnDisk(true, false)`, `Compile()`, `EnsureTinyGoInstalled()`

## Stages

### Stage 1: Crear `wasmbuild.go` — funcion testeable

**Archivo:** `client/wasmbuild.go`

```go
type WasmBuildArgs struct {
    Stdlib bool   // true = Go estandar modo "L", false = TinyGo modo "S"
}
```

Funcion `RunWasmBuild(args WasmBuildArgs) error`:

1. **Si no stdlib**: llamar `EnsureTinyGoInstalled()` y agregar al PATH (igual que cmd/client/main.go:65-76)
2. **Verificar input**: comprobar que `web/client.go` existe (`os.Stat`)
3. **Crear output dir**: `os.MkdirAll("web/public", 0755)`
4. **Generar script.js**:
   - Crear `Javascript{}` directamente (sin WasmClient)
   - Llamar `js.SetMode("S")` (o `"L"` si stdlib)
   - Llamar `js.SetWasmFilename("client.wasm")`
   - Llamar `js.GetSSRClientInitJS()` — retorna JS completo
   - Escribir resultado en `web/public/script.js`
5. **Compilar WASM**:
   - `cfg := NewConfig()` (defaults ya son `web` y `web/public`)
   - `w := New(cfg)`
   - `w.SetMode("S")` (o `"L"` si stdlib)
   - `w.SetBuildOnDisk(true, false)`
   - `w.SetLog(fmt.Println)`
   - `w.Compile()` — usa `DiskStorage.Compile()` -> `activeSizeBuilder.CompileProgram()`
   - Output: `web/public/client.wasm`

### Stage 2: Renombrar cmd/client -> cmd/wasmbuild

**Eliminar:** `client/cmd/client/` (directorio completo)
**Crear:** `client/cmd/wasmbuild/main.go`

```go
func main() {
    stdlib := flag.Bool("stdlib", false, "use Go standard compiler instead of TinyGo")
    flag.Parse()
    err := client.RunWasmBuild(client.WasmBuildArgs{Stdlib: *stdlib})
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### Stage 3: Tests

**Archivo:** `client/tests/wasmbuild_test.go`

Tests para `RunWasmBuild()` usando directorio temporal:
- **TestRunWasmBuild_GeneratesScriptJS_TinyGo**: modo default, verificar que `script.js` contiene signatures de `WasmExecTinyGoSignatures()` y footer `instantiateStreaming`
- **TestRunWasmBuild_GeneratesScriptJS_Stdlib**: modo stdlib, verificar signatures de `WasmExecGoSignatures()`
- **TestRunWasmBuild_FailsIfInputMissing**: sin `web/client.go`, retorna error
- **TestRunWasmBuild_CreatesOutputDir**: sin `web/public/`, verifica que lo crea

Nota: tests de generacion JS no requieren compilador instalado (solo verifican el `script.js`). Tests de compilacion completa requieren TinyGo/Go instalado.

### Stage 4: README y documentacion

**Crear:** `client/cmd/wasmbuild/README.md`
- Instalacion: `go install github.com/tinywasm/client/cmd/wasmbuild@latest`
- Uso default: `wasmbuild` — TinyGo modo S, genera `web/public/script.js` + `web/public/client.wasm`
- Uso stdlib: `wasmbuild -stdlib` — Go estandar modo L
- Requisito: archivo `web/client.go` debe existir

**Actualizar:** `client/README.md` — agregar enlace a `cmd/wasmbuild/`

## Notas de implementacion
- Reusar assets embebidos existentes via `Javascript` struct, no duplicar bytes
- `Javascript.GetSSRClientInitJS()` ya genera JS completo listo para `<script>` tag
- `DiskStorage.Compile()` ya crea el directorio de salida con `os.MkdirAll`
- Para los tests de JS: separar la generacion del JS (no necesita compilador) de la compilacion (necesita TinyGo)
- Los paths son fijos (`web/client.go`, `web/public/`), no configurables por flag
