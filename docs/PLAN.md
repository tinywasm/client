# PLAN — client: el `client.go` generado importa módulos que no están en el `go.mod` del proyecto → build inicial roto

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Parte de `tinywasm/docs/MCP_DAEMON_HARDENING_MASTER_PLAN.md`.
> Idioma: español (decisión del mantenedor). Autocontenido: el agente no tiene contexto previo.

## Prerequisito (correr primero)

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Todos los tests con `gotest` (nunca `go test` a secas).

---

## 0. Diagnóstico (evidencia real, 2026-07-09)

Proyecto mínimo (solo `go.mod` con `module demoproj` + `web/main.go`) iniciado
vía `start_development`. `CreateDefaultWasmFileClientIfNotExist`
(`generator.go:16`) genera `web/client.go` desde la plantilla embebida
(`templates/basic_wasm_client.md`) y dispara la compilación inmediata
(`generator.go:70`). Resultado:

```
CLIENT  Generated WASM source file at .../demoproj/web/client.go
CLIENT  Error compiling generated client: compilation failed: exit status 1
Output: web/client.go:29:2: no required module provides package github.com/tinywasm/dom; to add it:
        go get github.com/tinywasm/dom
web/client.go:30:2: no required module provides package github.com/tinywasm/fmt; ...
web/client.go:31:2: no required module provides package github.com/tinywasm/html; ...
```

El generador escribe código que **no puede compilar**: la plantilla importa
`tinywasm/dom`, `tinywasm/fmt` y `tinywasm/html` pero nadie agrega esos
requires al `go.mod` del proyecto. La experiencia de arranque de un proyecto
nuevo nace rota y el usuario/LLM debe adivinar tres `go get`.

## 1. Reglas de código (obligatorias)

- Este paquete corre server-side (tooling): stdlib permitida donde ya se usa.
- Strings repetidos → constantes tipadas: la lista de módulos que la plantilla
  importa vive en UNA constante/slice del paquete, junto a la plantilla — si
  la plantilla cambia sus imports, la lista se actualiza en el mismo commit
  (dejar comentario cruzado en ambos archivos).
- Errores se propagan; nada se traga en silencio.
- No overwrite: `CreateDefaultWasmFileClientIfNotExist` sigue sin tocar un
  `client.go` existente.

## 2. Etapa 1 — asegurar dependencias antes de compilar el archivo generado

En `generator.go`, tras generar `client.go` y ANTES de `store.Compile()`:

1. Declarar los módulos requeridos por la plantilla:

```go
// templateModules: módulos que importa templates/basic_wasm_client.md.
// Mantener sincronizado con los imports de la plantilla.
var templateModules = []string{
    "github.com/tinywasm/dom",
    "github.com/tinywasm/fmt",
    "github.com/tinywasm/html",
}
```

2. Para cada módulo ausente en el `go.mod` del proyecto, agregarlo. Evaluar en
   este orden (elegir el primero disponible y documentar la elección):
   - la interfaz `GoModHandler`/`devflow.GoModInterface` que ya circula por el
     ecosistema (grep en `devflow` por `GoMod`) si expone agregar requires;
   - si no, ejecutar `go get <mod>@latest` en el root del proyecto vía la
     abstracción de comandos existente en este paquete (grep cómo se invoca el
     compilador en `wasmbuild.go` y reutilizar ese mecanismo de exec/logging).
3. Si agregar la dependencia falla (sin red, etc.): loguear el error COMPLETO
   con la instrucción exacta (`go get ...`) y **no** intentar compilar — el
   error de deps es más accionable que el de compilación.
4. Solo con deps resueltas → `store.Compile()` como hoy.

## 3. Etapa 2 — el resultado del build inicial no puede ser ambiguo

Hoy `generator.go:70–72` loguea el error y sigue como si nada. Mantener el log
pero además propagar el estado: quien consuma este flujo (el daemon de `app`
loguea "Project ready" inmediatamente después) debe poder distinguir
build-verde de build-roto. Exponer el error de la compilación inicial en el
estado del `WasmClient` (campo/método consultable, p. ej. `LastBuildError()
error`) para que `app` lo consulte — sin romper la API existente.

## 4. Tests (con `gotest`)

1. Proyecto temporal con `go.mod` mínimo → generar → los requires de
   `templateModules` quedan en el `go.mod` → compilación inicial verde
   (si el entorno de test no tiene red, clasificar `env-blocked` y assertar
   al menos la modificación del `go.mod`).
2. `client.go` preexistente → no se toca ni se modifican deps.
3. Test guard: los imports `github.com/*` extraídos de
   `templates/basic_wasm_client.md` ⊆ `templateModules` (así la plantilla no
   puede desincronizarse en silencio).

## 5. Etapa 3 — documentación

- `README.md`/`docs/ARCHITECTURE.md`: documentar que la generación del client
  default garantiza sus propias dependencias en el `go.mod` del proyecto.

## 6. Criterios de aceptación

1. `gotest ./...` verde.
2. Proyecto mínimo nuevo: build inicial verde sin intervención manual.
3. Fallo al resolver deps → mensaje accionable, sin compilación fantasma.
4. Test guard de sincronía plantilla ↔ `templateModules` presente.

## 7. Tabla de etapas

| # | Etapa | Archivos | Gate |
|---|-------|----------|------|
| 1 | Deps garantizadas antes de compilar | `generator.go` (+constante) | tests §4.1–4.2 |
| 2 | Estado de build inicial consultable | `client.go`/`wasmbuild.go` | compila, API no rota |
| 3 | Guard de sincronía + docs | test nuevo, `README.md` | §4.3 verde |
