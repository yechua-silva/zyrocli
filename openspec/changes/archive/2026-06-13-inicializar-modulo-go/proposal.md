# Proposal: Inicializar módulo Go y estructura base del proyecto

## Intent

ZyroCLI existe como scaffold sin código compilable: no `go.mod`, no archivos Go, no `go build`. Se necesita la estructura base para desbloquear el desarrollo de paquetes internos.

## Scope

### In Scope
- `go.mod` + `go.sum` — módulo `github.com/secko/zyrocli`, Go 1.22
- `cmd/zyrocli/main.go` — Cobra root command
- 7 paquetes `internal/` con stubs compilables: `scheduler`, `handoff`, `skilladvisor` (3 files), `spec` (2 files), `context`, `apply`, `test` (2 files)
- `.gitignore` (estándar Go), `handoff.yaml` (ejemplo), `zyro-skill-overrides.yaml` (ejemplo)

### Out of Scope
- Lógica de negocio real, tests, comandos CLI más allá del root, integración MCP real, lint/CI/CD

## Capabilities

### New Capabilities
- `project-scaffold`: estructura base del proyecto Go (go.mod, cmd/, 7 paquetes internal/ stubs compilables, configs de ejemplo)

### Modified Capabilities
None

## Approach

"All at once" — single change:
1. `go mod init github.com/secko/zyrocli`
2. `go get github.com/spf13/cobra gopkg.in/yaml.v3 `
3. Crear `cmd/zyrocli/main.go` con Cobra root
4. Crear 7 stubs internal/ compilables (structs vacíos + placeholders `// TODO`)
5. Crear `.gitignore`, `handoff.yaml`, `zyro-skill-overrides.yaml`
6. Verificar con `go vet ./...`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod` | New | Module + deps |
| `go.sum` | New | Checksums |
| `cmd/zyrocli/main.go` | New | Cobra entry point |
| `internal/scheduler/scheduler.go` | New | DAG executor stub |
| `internal/handoff/payload.go` | New | Handoff parser stub |
| `internal/skilladvisor/*.go` | New | 3 stubs |
| `internal/spec/*.go` | New | 2 stubs |
| `internal/context/bridge.go` | New | MCP bridge stub |
| `internal/apply/runner.go` | New | Task runner stub |
| `internal/test/*.go` | New | 2 stubs |
| `.gitignore` | New | Go standard |
| `handoff.yaml` | New | Example input |
| `zyro-skill-overrides.yaml` | New | Example config |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Go 1.22+ not installed | Low | `go version` check first |
| Filesystem restrictions | Low | All writes via bash |
| Dep version conflicts | Low | Use latest stable releases |

## Rollback Plan

`rm -rf cmd/ internal/ go.mod go.sum .gitignore handoff.yaml zyro-skill-overrides.yaml`

## Dependencies

- Go 1.22+ en `$PATH`
- `github.com/spf13/cobra`, `gopkg.in/yaml.v3`

## Success Criteria

- [ ] `go vet ./...` pasa sin errores
- [ ] `go build ./...` produce binario
- [ ] `zyrocli --help` muestra root command
- [ ] Cada stub internal/ tiene al menos un tipo exportado o función
