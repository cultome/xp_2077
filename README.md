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

### Validación esperada

El script imprime un resumen con:

- total de registros extraídos por fuente
- total persistido por tabla en SQLite
- cantidad de `issue_node_id` duplicados entre fuentes
- rango temporal de `updated_at` detectado
