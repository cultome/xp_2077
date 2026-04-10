# xp_2077

CLI/TUI para visualizar ranking de XP.  
Incluye un script independiente para validar extracción de datos desde GitHub antes de integrarlo al UI.

## Script de extracción GitHub a SQLite

Este script extrae datos desde:

- GitHub Project v2 (items cuyo `content` sea `Issue`)
- Issues de un repositorio (filtra pull requests)

Luego persiste:

- payload crudo de Project v2 en `project_items_raw`
- payload crudo de repo issues en `repo_issues_raw`
- versión normalizada en `issues_normalized`
  - `issue_body`
  - `xp_base`
  - `planned_start_date`
  - `planned_end_date`
  - `real_end_date`
  - `xp_final`

### Regla XP persistida

`xp_final` se calcula solo cuando existen los 4 campos de Project v2:

- `XP`
- `fecha programada de inicio`
- `fecha programada de fin`
- `fecha real de fin`

Fórmula:

- `delta_days = planned_end_date - real_end_date`
- `delta_pct = abs(delta_days) / (planned_end_date - planned_start_date)`
- si `delta_days > 0`: `xp_base + (xp_base * delta_pct)`
- si `delta_days < 0`: `xp_base - (xp_base * delta_pct)`
- si `delta_days == 0`: `xp_base`
- redondeo a 1 decimal y clamp mínimo a `0`

### Variables de entorno

- `GITHUB_TOKEN` (requerida)
- `GITHUB_OWNER` o `GITHUB_ORG` (requerida)
- `GITHUB_REPO` (requerida, acepta `repo` o `owner/repo`)
- `GITHUB_PROJECT_NUMBER` (requerida, número de Project v2)
- `OUTPUT_DB` (opcional, default `./tmp/github_extract.db`)

### Ejecutar

```bash
go run ./cmd/github_extract \
  -owner "$GITHUB_OWNER" \
  -repo "$GITHUB_REPO" \
  -project "$GITHUB_PROJECT_NUMBER" \
  -db "${OUTPUT_DB:-./tmp/github_extract.db}"
```

También puedes omitir flags y usar variables de entorno.

## Ejecutar cliente TUI sin extracción

Si ya tienes datos en SQLite y quieres abrir el cliente sin volver a consultar GitHub:

```bash
go run . -skip-extract
```

Con `-skip-extract` la app omite validaciones de variables GitHub y salta directo al Home usando los datos existentes en `OUTPUT_DB`.

### Validación esperada

El script imprime un resumen con:

- total de registros extraídos por fuente
- total persistido por tabla en SQLite
- cantidad de `issue_node_id` duplicados entre fuentes
- rango temporal de `updated_at` detectado
