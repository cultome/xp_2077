# xp_2077

TUI para visualizar un ranking de **XP** por persona, calculado a partir de tareas en GitHub.
Al arrancar, la app extrae datos de GitHub a SQLite y luego muestra el leaderboard.

## Qué extrae

- GitHub Project v2 (items cuyo `content` sea `Issue`)
- Issues de un repositorio (filtra pull requests)

Persiste:

- payload crudo de Project v2 en `project_items_raw`
- payload crudo de repo issues en `repo_issues_raw`
- versión normalizada en `issues_normalized`
  - `issue_body`
  - `xp_base`
  - `planned_start_date`
  - `planned_end_date`
  - `real_end_date`
  - `xp_final`

### Reglas XP persistidas

Una tarea solo aparece en el ranking si tiene `xp_final` **y** `real_end_date`.
`xp_final` se calcula por una de dos reglas según el origen:

**Project v2** — requiere los 4 campos (`XP`, `fecha programada de inicio`,
`fecha programada de fin`, `fecha real de fin`; aceptan alias `Implementacion *`):

- `delta_days = planned_end_date - real_end_date`
- `delta_pct = abs(delta_days) / (planned_end_date - planned_start_date)`
- si `delta_days > 0`: `xp_base + (xp_base * delta_pct)`
- si `delta_days < 0`: `xp_base - (xp_base * delta_pct)`
- si `delta_days == 0` (o plan de un solo día): `xp_base`
- redondeo a 1 decimal y clamp mínimo a `0`

**Repo issue** — solo si el título empieza con `[Special Tasks for Aleph] `, el `Status`
es `done`, y existen `Story Points` + `Priority` (P0=2, P1=1.5, P2=1) + `Due Date`:

- `xp_final = round(story_points * multiplier, 1)`

### Variables de entorno

- `GITHUB_TOKEN` (requerida) — debe poder leer **todos** los repos que el Project referencia;
  si no, esos issues se omiten (se reportan en Home).
- `OUTPUT_DB` (opcional, default `./tmp/github_extract.db`)

Configuración fija de GitHub (en código, `internal/extract/extract.go`):

- owner/org: `aleph-ri`
- repo: `advance`
- project number: `12`

## Ejecutar

```bash
go run .            # extrae de GitHub y abre el leaderboard
```

Durante la carga se muestra el progreso de extracción. En Home se reporta cuántas cards de
issue se omitieron por falta de acceso al repo.

### Sin extracción (usar SQLite existente)

```bash
go run . -skip-extract
```

Con `-skip-extract` la app omite la validación de variables de GitHub y salta directo al Home
usando los datos existentes en `OUTPUT_DB`.
