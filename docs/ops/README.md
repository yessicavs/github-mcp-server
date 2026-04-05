# Ops Documentation Index
> `yessicavs/github-mcp-server` · Infraestructura MCP de Ops@growthxy.com
> Actualizado: 2026-04-05

---

## Documentos activos

| Archivo | Descripción | Última actualización |
|---|---|---|
| [github-mcp-audit-comparativo.md](github-mcp-audit-comparativo.md) | Auditoría comparativa: `shared-github-mcp-server-1` vs `github-mcp-proxy` — tools, comportamiento, gaps | 2026-04-05 |
| [test-patch.md](test-patch.md) | Archivo de test para verificar endpoints `/github-patch` y `/github-append` | 2026-04-05 |

### Auditorías históricas (Workers Observability)

En `/mnt/user-data/outputs/` de la sesión original:
- `neo4j-mcp-audit.md` — auditoría de `mcp-neo4j-cypher` (v1)
- `github-mcp-audit.md` — auditoría de `shared-github-mcp-server-1` (v1)
- `github-mcp-audit-v2.md` — corrige arquitectura DO+PartyKit (v2)
- `github-mcp-audit-v3.md` — añade hallazgos de Cloudflare API (v3)

---

## Stack actual

```
Claude.ai web
    │  OAuth (registro → autorización → token)
    ▼
github-mcp-proxy.ops-e1a.workers.dev   ← github-mcp-proxy v3.0
    │  Bearer <PAT>
    ▼
api.githubcopilot.com/mcp/             ← github/github-mcp-server v0.32.0
    │
    ▼
GitHub API  (80+ tools)
```

| Recurso | Valor |
|---|---|
| Worker | `github-mcp-proxy` (Cloudflare, cuenta `Ops@growthxy.com`) |
| KV | `github-mcp-proxy-OAUTH` (`20cb14eff6cf4a9cbc7d0119018f0876`) |
| URL MCP | `https://github-mcp-proxy.ops-e1a.workers.dev/mcp` |
| Versión Worker | v3.0.0 (2026-04-05) |
| Upstream | `api.githubcopilot.com/mcp/` → `github/github-mcp-server` v0.32.0 |

---

## Endpoints de edición de documentos

Todos requieren `Authorization: Bearer <oauth_access_token | github_pat>`.

### `POST /github-read`
Lee un archivo como texto plano UTF-8 sin que el base64 pase por el transporte MCP.
Soporta archivos >1MB automáticamente vía `download_url`.

```json
{ "owner": "...", "repo": "...", "path": "docs/ops/README.md", "branch": "main" }
```

### `POST /github-patch` (str_replace)
Reemplaza texto con validación estricta de unicidad. Un solo commit.
Soporta modo single y modo multi-patch (array).

```json
// Single patch
{ "owner": "...", "repo": "...", "path": "...",
  "old_str": "texto exacto a reemplazar",
  "new_str": "texto nuevo",
  "message": "docs: actualiza sección X" }

// Multi-patch — múltiples cambios en un solo commit
{ "owner": "...", "repo": "...", "path": "...",
  "patches": [
    { "old_str": "sección A original", "new_str": "sección A nueva" },
    { "old_str": "sección B original", "new_str": "sección B nueva" }
  ],
  "message": "docs: actualización semanal" }
```

Errores posibles:
- `not_found` (422) — `old_str` no encontrado; incluye stats del archivo y hint
- `ambiguous` (422) — `old_str` aparece >1 vez; incluye posiciones y contexto
- `conflict` (409) — SHA mismatch; el archivo fue editado concurrentemente

**CRLF:** normaliza `\r\n` → `\n` automáticamente antes del matching.

### `POST /github-append`
Añade contenido al final del archivo con separador inteligente (añade `\n` si el archivo no termina en newline).

```json
{ "owner": "...", "repo": "...", "path": "docs/ops/changelog.md",
  "content": "\n## 2026-04-05\n- entrada nueva",
  "message": "docs: entrada changelog 2026-04-05" }
```

### `POST /github-search`
Busca un string dentro de un archivo. Devuelve matches con líneas de contexto.

```json
{ "owner": "...", "repo": "...", "path": "docs/ops/audit.md",
  "query": "storage leak",
  "context_lines": 3,
  "max_matches": 20,
  "case_sensitive": false }
```

---

## Patrón de frontmatter recomendado

Todos los documentos de esta carpeta deben incluir un header de metadatos para facilitar la orientación en sesiones nuevas sin tener que leer el historial de commits:

```markdown
# Título del documento
> `contexto/repo` · descripción breve
> Actualizado: YYYY-MM-DD · commit `abc1234`
```

En cada sesión de actualización, actualizar la línea `Actualizado:` con `/github-patch` como primer paso.

---

## Workflow incremental recomendado

### Sesión nueva sobre un documento existente

1. **Orientarse** — `get_file_contents` o `/github-read` para ver el estado actual y su SHA
2. **Localizar** — `/github-search` para encontrar la sección exacta sin leer el documento entero
3. **Parchear** — `/github-patch` con `old_str` suficientemente único (incluir encabezado de sección)
4. **Actualizar metadatos** — `/github-patch` sobre la línea `Actualizado:` del frontmatter
5. **Verificar** — `get_file_contents` para confirmar el resultado

### Crear un documento nuevo

1. Crear con `MCP_GITHUB:create_or_update_file` o `push_files` (varios archivos en un commit)
2. Incluir el frontmatter con la fecha del día
3. Añadir entrada al índice (`docs/ops/README.md`) con `/github-append` o `/github-patch`

### Límites del transporte MCP

| Tamaño del archivo | Operación recomendada |
|---|---|
| < 30 KB | `get_file_contents` + `create_or_update_file` directamente |
| 30 – 70 KB | `/github-patch` (evita transportar el archivo completo) |
| > 70 KB | `/github-read` + `/github-patch` obligatorio |
| > 1 MB | `/github-read` automáticamente usa `download_url` |
