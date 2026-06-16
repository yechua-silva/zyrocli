# Proposal: Implementar parser de handoff.yaml + comando init

## Intent

ZyroCLI necesita un parser completo de `handoff.yaml` que valide el contrato v2.0 definido en AGENT.md, soporte pipe desde Holdin Admin via stdin (`-`), y exponga el comando `zyrocli init <path>`. Sin esto, el pipeline de 4 fases no puede arrancar.

## Scope

### In Scope
- Reestructurar `payload.go` con structs alineados al contrato v2.0 (Source, Project, ValidatedIdea, UserStory, MVP, Governance, Testing, Limits)
- `parser.go` — `Parse(path string) (*Payload, error)` con soporte de archivo real y stdin (`"-"`)
- `validate.go` — `Validate(p *Payload) error` con reglas de campos requeridos estrictos
- `payload_test.go` — 8+ casos table-driven (archivo valido, stdin, version invalida, campos faltantes, etc.)
- Comando `zyrocli init <path>` (o `-` para stdin) en `cmd/zyrocli/main.go`
- Actualizar `handoff.yaml` de ejemplo a v2.0

### Out of Scope
- Notificacion a Holdin Admin (comunicacion inversa)
- Integracion MCP real
- Comandos adicionales (`run`, `phase`, `skill-advisor`, etc.)
- Soporte de handoff v1.0

## Capabilities

### New Capabilities
- `handoff-parser`: parser + validator de handoff.yaml v2.0, soporte file/stdin

### Modified Capabilities
- None

## Approach

1. Reemplazar structs en `payload.go` con los del contrato v2.0, tags `yaml` completas
2. `parser.go`: `os.ReadFile` para path real, `io.ReadAll(os.Stdin)` para `"-"`, `yaml.Unmarshal`
3. `validate.go`: checklist de campos requeridos, version exacta `"2.0"`, `source.system != ""`
4. Comando: registrar `initCmd` en `main.go` con `cobra.Command`, args validados
5. Tests: table-driven con `t.TempDir()` para archivos temporales
6. `handoff.yaml` raiz: reemplazar con contrato v2.0 legible

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/handoff/payload.go` | Modified | Structs v2.0 completos |
| `internal/handoff/parser.go` | New | Parse + stdin support |
| `internal/handoff/validate.go` | New | Business rule validation |
| `internal/handoff/payload_test.go` | New | Table-driven tests |
| `cmd/zyrocli/main.go` | Modified | Init subcommand |
| `handoff.yaml` | Modified | Ejemplo a v2.0 |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Struct mismatch con contrato real | Low | Validacion estricta + tests contratan contra el schema exacto |

## Rollback Plan

```bash
git checkout -- internal/handoff/ cmd/zyrocli/main.go handoff.yaml
```

## Dependencies

- `gopkg.in/yaml.v3` (ya instalada en go.mod)
- `github.com/spf13/cobra` (ya instalada)

## Success Criteria

- [ ] `zyrocli init handoff.yaml` parsea archivo valido y muestra resumen
- [ ] `echo "..." \| zyrocli init -` parsea stdin correctamente
- [ ] Validacion rechaza archivos con campos requeridos faltantes
- [ ] Validacion rechaza version distinta de `"2.0"`
- [ ] 8+ tests table-driven pasan (`go test ./internal/handoff/`)
- [ ] `go vet ./...` sin errores
- [ ] `go build ./...` exitoso
