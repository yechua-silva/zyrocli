# Deploy Conventions

> Convenciones para CI/CD y estructura de código del proyecto ZyroCLI.
> Última actualización: Junio 2026 · Runtime objetivo: Node 24

## Versiones canónicas de Actions

| Action | Versión | Node runtime | Runner mínimo |
|--------|---------|-------------|---------------|
| `actions/checkout` | `@v7` | Node 24 | v2.329.0+ |
| `actions/setup-go` | `@v6` | Node 24 | v2.327.1+ |
| `actions/setup-node` | `@v6` | Node 24 | v2.327.1+ |
| `goreleaser/goreleaser-action` | `@v7` | Node 24 | v2.327.1+ |

## Node.js

- **LTS actual (2026)**: Node 22 (activo hasta Octubre 2026).
- Usar `node-version: '22'` en workflows.

## GoReleaser

- `version: 2` (formato v2, obligatorio desde GoReleaser v2).
- `snapshot.version_template` en vez de `snapshot.name_template` (deprecado desde v2.2).
- `archives.formats` en vez de `archives.format` (deprecado desde v2.6).
- `archives.ids` en vez de `archives.builds` (deprecado desde v2.8).
- Ejecutar `goreleaser check` para validar config.

## Go

Orden obligatorio en todo `.go` file:
```
//go:build <constraint>    ① (opcional)
package <name>             ② (obligatorio)
import (                   ③ (opcional, pero debe ir ANTES que cualquier declaración)
    "..."
)
// declaraciones           ④ tipos, vars, consts, funcs
```

## release.yml

```yaml
- uses: actions/checkout@v7
  with:
    fetch-depth: 0
    ref: ${{ github.ref }}
- uses: actions/setup-go@v6
- uses: goreleaser/goreleaser-action@v7
  with:
    version: '~> v2'
```

## Historial de migración

| Fecha | Cambio |
|-------|--------|
| Feb 2026 | goreleaser-action@v7: Node 24, ESM |
| Jun 2026 | checkout@v7: ESM, bloqueo fork PRs |
