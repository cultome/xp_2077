# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> Idioma: el equipo trabaja en español. Comentarios de docs en español; código y tests en inglés.

## Qué es esto

TUI (Bubble Tea) que muestra un leaderboard de **XP** por persona, calculado a partir de
tareas en GitHub. Al arrancar, la app **extrae** datos de GitHub a SQLite y luego renderiza
el ranking. El contexto de GitHub está **fijo en código** (`internal/extract/extract.go`):

- org/owner: `aleph-ri`, repo: `advance`, GitHub Project v2 número `12` (título "Aleph SDLC v2").

## Comandos

```bash
go run .                 # arranca la TUI (extrae de GitHub y muestra el leaderboard)
go run . -skip-extract   # arranca usando SOLO el SQLite existente (no consulta GitHub)
go build -o bin/xp_2077 . # compila la app
go test ./...            # toda la suite
go test ./internal/github/ -run TestClassifyProjectNode  # un solo test (por nombre/regex)
go vet ./... && gofmt -l internal/ main.go               # estática + archivos mal formateados
```

`make build` / `make run` / `make run-skip-extract` / `make test` / `make fmt` también funcionan.
La extracción vive **dentro de la app** (`main.go` → `internal/ui` → `internal/extract`); ya no
existe el comando `cmd/github_extract` (se eliminó en commit 28e89ce).

### Variables de entorno

- `GITHUB_TOKEN` (requerida; la app lo valida en la pantalla env-check antes de extraer).
- `OUTPUT_DB` (opcional, default `./tmp/github_extract.db`). `tmp/` está en `.gitignore`.

## Arquitectura (big picture)

Flujo: `main.go` abre SQLite y lanza la TUI → la TUI corre la extracción → consulta la BD.

```
main.go ─► internal/ui (Bubble Tea)
              │  rutas: splash → env_check → loading → home → detail → issue_detail
              │  (-skip-extract salta splash → home con la BD existente)
              ▼
        internal/extract  (extract.Run: pipeline de 6 stages + Tracker de progreso)
              ▼
        internal/github   (Client: dos fuentes de datos)         internal/store (SQLite)
              ▼                                                        ▲
        internal/domain   (Repository, TaskXP, UserXP, DateRange) ────┘
```

### Dos fuentes de extracción (clave)

`extract.Run` junta **dos** orígenes en la tabla `issues_normalized` con columna `source`:

1. **`project_v2`** — `Client.FetchProjectV2IssuesWithProgress`: GraphQL paginado sobre los
   items del Project #12. `source_record_id` = ID del **item del project** (no del issue).
   El XP sale de campos del project (`XP`, `Implementacion Inicio/Fin/Fin Real`).
2. **`repo_issue`** — `Client.FetchRepoIssuesWithProgress`: REST paginado de issues del repo
   **`advance` únicamente** (hardcoded), filtra PRs, y enriquece cada issue con sus campos de
   project vía `fetchProjectFieldsForIssueNodeIDs` (GraphQL `nodes(ids:)`).

El mismo issue suele existir en ambas fuentes (≈246 issues de advance están en ambas); se
guardan como filas separadas y, por diseño, el XP se calcula por una sola de las dos rutas.

### Reglas de XP (`internal/github/types.go`)

Una tarea **solo aparece** en leaderboard/detalle si tiene `xp_final` **y** `real_end_date`
(ver queries en `store/sqlite.go`). Hay dos cálculos distintos:

- **Proyecto** (`parseProjectXPFields`): requiere `XP` + 3 fechas (con alias, p.ej. "fecha
  programada de fin" ≡ "implementacion fin"). Penaliza/bonifica por desviación vs fecha
  planeada; clamp a 0. Tareas de **un solo día** (inicio == fin) acreditan el XP base.
- **Repo** (`parseRepoIssueXPFields`): solo si el título empieza con `[Special Tasks for
  Aleph] `, Status = done, y existen `Story Points` + `Priority` (P0=2, P1=1.5, P2=1) + `Due Date`.

El matcheo de nombres de campo normaliza acentos/mayúsculas/espacios (`normalizeFieldName`).

### Persistencia (`internal/store`)

Tres tablas: `project_items_raw`, `repo_issues_raw`, `issues_normalized` (PK
`(source, source_record_id)`). Todo es **UPSERT (`ON CONFLICT`) que nunca borra** → filas de
issues eliminados/movidos en GitHub **persisten** (los conteos en BD pueden superar a los de
GitHub en vivo). El schema base está en `schema.sql`; columnas/índices extra se añaden en
runtime (`ensureIssuesNormalizedColumns/Indexes`) — los ALTER son aditivos e idempotentes.

## Gotchas críticos (aprendidos depurando extracción)

- **El Project #12 abarca varios repos** (advance, alliance, webapp-v1, aliado-webapp,
  AliadoRN, scraper, …). El `GITHUB_TOKEN` **debe poder leer TODOS** esos repos (Issues:Read).
  Si no, esos issues llegan con `content: null` y se **omiten**. Con fine-grained PAT, dar
  acceso a "All repositories" del org evita huecos cuando el project gana repos nuevos.
- **Tolerancia a errores parciales de GraphQL** (`fatalGraphErrors` en `client.go`): GitHub
  devuelve `data` parcial + errores por-item (`FORBIDDEN` en el `content` de una card de un
  repo sin acceso). Esos se ignoran y se continúa; solo abortan los errores de nivel superior.
  NO revertir a "abortar ante cualquier error" — rompe la extracción ante una sola card vetada.
- **Contador de cards omitidas** (`ProjectFetchStats` → `extract.Result` → Home): distingue
  `InaccessibleIssues` (issues no leídos = pérdida real, se muestran en rojo) de `NonIssues`
  (PRs/drafts, descarte benigno). `classifyProjectNode` usa el `type` del item del project.

## Estado de desarrollo (mantener al día entre sesiones)

> Actualiza esta sección al final de cada sesión: qué se hizo, qué quedó pendiente.

**Estado actual:** extracción funcionando end-to-end (≈941 filas normalizadas, multi-repo).
El issue 1740 (repo `alliance`) y ~27 tareas que antes no se veían ahora sí aparecen.

**Hecho recientemente (sesión 2026-06-18):**
- Multi-repo resuelto **ampliando el scope del token** (no se hizo extracción "solo cards").
- `client.go`: `fatalGraphErrors` tolera errores `FORBIDDEN` por-item; `classifyProjectNode`
  + `ProjectFetchStats` cuentan cards omitidas; query del project pide `type` del item.
- `types.go`: `parseProjectXPFields` ya no descarta tareas de un solo día (solo rechaza
  duración negativa).
- UI: contador de cards omitidas en Home (`view_home.go: extractionSummaryLine`).
- Tests nuevos en `types_test.go` (classify, fatalGraphErrors, same-day, duración negativa).
- Limpieza de `Makefile` y `README.md`: removidas las referencias al `cmd/github_extract`
  eliminado; agregado target `run-skip-extract`.
- TUI cyberpunk (ámbar + acentos neón): paleta semántica en `styles.go`
  (Error rojo `#FF2A6D`, Success verde `#00FF9C`, Link cian `#00F0FF`); helpers en
  `animation.go` (`xpBar`/`xpFillCount`, `glitch`, `decryptReveal`, `scanline`, `hudClock`);
  HUD global en `screen()`; leaderboard custom con barras de XP por tier en `view_home.go`
  (la `userTable` queda solo como estado de cursor); glitch/reveal en headers y splash;
  filtro de usuario, orden (`s`), presets de fecha (`p`), re-extraer (`ctrl+e`), abrir issue
  en navegador (`o`, `openurl.go`). Tests en `internal/ui/ui_extras_test.go`.

**Pendientes / deuda conocida:**
- La tabla de la vista DETALLE (`detailTable`) hace wrap de borde cuando es más ancha que el
  frame (preexistente, no relacionado con el HUD) → ajustar anchos de columnas/Panel.
- Ruta `repo_issue` está fija a `advance`; issues `[Special Tasks for Aleph]` en otros repos
  no reciben la regla de XP de repo (sí la de proyecto si tienen los campos).
- UPSERT no purga issues borrados/movidos → considerar limpieza/marcado de obsoletos.
- `internal/mock/pipeline.go` no está gofmt-clean (preexistente).
- Se evaluó (y descartó por ahora) extraer "solo cards" del project usando snapshots de
  campos (Title/Assignees/XP/fechas), que funcionaría aun sin acceso al repo del issue.
