---
message: "feat: serve the wasm bundle with router.PublicAsset — public by construction"
---

> Este plan se despacha vía el flujo CodeJob. Ver skill: agents-workflow.
> Orquestado por `tinywasm/docs/PUBLIC_ASSETS_MASTER_PLAN.md` — **Fase C2**.
> Autocontenido: el agente que lo ejecuta no tiene contexto previo.
>
> **COMPUERTA:** requiere `github.com/tinywasm/router` **v0.1.7+** (ya publicado), que es
> quien declara `PublicAsset`. Sube `go.mod` primero. Si `router.Router` no tiene
> `PublicAsset`, PARA y reporta.
> No dependes de `server`: este repo solo consume la **interfaz**. Corre en paralelo con él.

# PLAN — `client`: el bundle wasm se registra con `PublicAsset`

**En una frase:** los **dos** storages (memoria y disco) registran `/client.wasm` con
`r.Get(...)`, lo que lo deja **privado** → el navegador recibe **403** y la app no arranca.
Pasan a `r.PublicAsset(...)`.

Cambio pequeño y mecánico. Lo que importa es **por qué**, para que no se deshaga.

---

## El problema (contexto, ya diagnosticado — no lo reabras)

`tinywasm/router` es **privado por defecto**: una ruta que no declara `Public()` ni
`Requires()` deniega a quien no tiene identidad. Correcto, y no se toca.

El navegador que pide `/client.wasm` **es anónimo** — la app ni siquiera ha arrancado, no
hay sesión posible. Registrado con `r.Get(...)` a secas, el bundle respondía **403**. Junto
con el mismo fallo en `assetmin`, el resultado era que **ningún proyecto renderizaba**: build
en verde, página que dice `Forbidden`.

**Por qué nadie lo detectó:** el mock del router copiaba el `RouteInfo` **por valor** al
registrar, así que un `.Public()` encadenado después mutaba otra copia. Para el mock **toda
ruta era privada, siempre**. Corregido en `router` v0.1.6; por eso ahora sí se puede afirmar
en un test.

## La decisión (no la reabras)

Se descartó añadir `.Public()`: es un *"no olvides llamar a X"*, y el olvido **no falla en
compilación ni hace ruido**. `router` v0.1.7 lo cierra con tipos:

```go
// UN archivo, UNA ruta. Público por construcción. NO devuelve Route: no hay permiso
// que colgarle → no se puede olvidar abrirlo, ni cerrarlo por error.
PublicAsset(path string, h HandlerFunc)
```

---

## Paso 1 — `go.mod`

Sube `github.com/tinywasm/router` a **v0.1.7+**. Verifica que `PublicAsset` existe antes de seguir.

## Paso 2 — los DOS storages

**Son dos sitios distintos, y es fácil arreglar solo uno.** Ambos sirven el mismo archivo:

| Archivo | Tipo | Línea aprox. |
|---|---|---|
| `http.go` | `MemoryStorage.RegisterRoutes` | `r.Get(routePath, …)` ~21 |
| `storage.go` | `DiskStorage.RegisterRoutes` | `r.Get(routePath, …)` ~88 |

En los dos, cambia `r.Get(routePath, func(ctx router.Context) {…})` por
`r.PublicAsset(routePath, func(ctx router.Context) {…})`.

**El cuerpo del handler no cambia** — ni el 503 de "WASM compiling...", ni la lectura de
disco, ni las cabeceras. `PublicAsset` no devuelve `Route`, así que no hay nada que
encadenar detrás.

## Paso 3 — el guard (escríbelo: hoy no existe)

Con `github.com/tinywasm/router/mock`, y **para los dos storages** (el test existente
`tests/in_memory_test.go` ya monta un `mock.Router`; sigue ese patrón):

```go
for _, route := range r.Routes() {
    if !route.Public {
        t.Errorf("%q privada → el navegador recibe 403 y el wasm nunca carga", route.Path)
    }
}
```

Comprueba que **de verdad caza el bug**: vuelve un `PublicAsset` a `r.Get`, confirma que el
test se pone rojo señalando esa ruta, y restáuralo. Un test que pasa en ambos casos no vale
de nada — es exactamente el error que dejó pasar este bug la primera vez.

---

## ⚠️ Anti-footguns (NO hagas esto)

- **NO añadas `.Public()`.** Si lo estás buscando, quieres `PublicAsset`.
- **NO arregles solo un storage.** Son dos, memoria y disco.
- **NO toques la lógica de compilación wasm**, ni el 503 de "compilando", ni las cabeceras.
- **NO toques `assetmin` ni `server`**: migran con sus propios planes.
- Nunca ejecutes `gopush` ni `codejob`.

## Criterios de aceptación

```bash
grep -rn 'r\.Get(' http.go storage.go   # → vacío
grep -rn '\.Public()' .                  # → vacío
gotest                                    # verde, con el guard nuevo
```

## Al cerrar

Anota en `AGENTS.md`: *"el bundle wasm se registra con `router.PublicAsset` — es público por
construcción; nunca `Get(...).Public()`"*. Luego **borra este `docs/PLAN.md`**.
