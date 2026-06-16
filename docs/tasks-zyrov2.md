# Tareas Atómicas — ZyroAgentCLI v2

> **Fecha:** 2026-06-15  
> **Basado en:** `docs/design-zyrov2.md` + `docs/spec-zyrov2.md` + `docs/roadmap-integrado.md`  
> **Pipeline:** SDD Fase 3 (Task Breakdown)  
> **Total tareas:** 72  

---

## Sprint 0: CLI Inteligente
### Dependencias: Ninguna

#### T-0.1: Comando `zyro setup` (SetupCmd + flags)
- **Archivos:** `cmd/zyrocli/cmd/setup.go` (nuevo, ~60 líneas)
- **Depende de:** Nada
- **Descripción:** Crear el comando Cobra `SetupCmd` con flags `--dry-run`, `--verbose`, `--skip-go`, `--target-dir`, `--json`. Implementar `SetupFlags` struct y función `runSetup` con esqueleto que llame a detectOS, CheckAll, instaladores secuenciales. Registrar en `init()`.
- **Criterio:** `zyro setup --help` muestra los 5 flags correctamente. `go vet ./cmd/zyrocli/cmd/` pasa sin errores.

#### T-0.2: Comando `zyro doctor` (DoctorCmd + flags)
- **Archivos:** `cmd/zyrocli/cmd/doctor.go` (nuevo, ~50 líneas)
- **Depende de:** Nada
- **Descripción:** Crear el comando Cobra `DoctorCmd` con flags `--fix`, `--dry-run`, `--verbose`, `--json`. Implementar `DoctorFlags` struct y función `runDoctor`. Registrar en `init()`.
- **Criterio:** `zyro doctor --help` muestra los 4 flags. `runDoctor` imprime reporte básico.

#### T-0.3: Registrar comandos en main.go
- **Archivos:** `cmd/zyrocli/main.go` (modificar, +2 líneas)
- **Depende de:** T-0.1, T-0.2
- **Descripción:** Agregar `rootCmd.AddCommand(cmd.SetupCmd)` y `rootCmd.AddCommand(cmd.DoctorCmd)` en `main.go`.
- **Criterio:** `zyro --help` lista `setup` y `doctor`. `go build ./cmd/zyrocli` compila.

#### T-0.4: Simplificar install.sh
- **Archivos:** `scripts/install.sh` (modificar, ~30 líneas)
- **Depende de:** T-0.3
- **Descripción:** Simplificar `install.sh` para que sea un pipeline curl que: (1) verifique OS, (2) descargue/compile `zyrocli`, (3) ejecute `zyro setup`. Eliminar lógica duplicada de instalación de HelixDB y Go build (ahora `zyro setup` lo hace).
- **Criterio:** `bash scripts/install.sh` descarga/compila zyrocli y luego llama a `zyro setup --dry-run`.

#### T-0.5: Tipos e interfaz DependencyChecker (check.go parte 1)
- **Archivos:** `internal/setup/check.go` (nuevo, ~100 líneas)
- **Depende de:** T-0.1
- **Descripción:** Definir tipos: `Dependency` enum (DepGo, DepUV, DepHelixDB, DepDocker, DepGit), `Status` enum (StatusOK, StatusWarning, StatusError, StatusMissing), `CheckResult` struct con Name/Status/Version/Detail/Fix, `PlatformInfo` struct con OS/Arch/HomeDir/LocalBinDir/ConfigDir. `Checker` interface con CheckAll/CheckOne/Platform. `DependencyChecker` struct con constructor `NewDependencyChecker()`. `detectOS()` para detectar linux/darwin + arquitectura.
- **Criterio:** `NewDependencyChecker().Platform()` devuelve `PlatformInfo` correcta. Tests unitarios pasan.

#### T-0.6: Métodos de verificación individual (check.go parte 2)
- **Archivos:** `internal/setup/check.go` (modificar, ~100 líneas)
- **Depende de:** T-0.5
- **Descripción:** Implementar 5 métodos de verificación: `checkGo()` (exec `go version`), `checkUV()` (exec `uv --version`), `checkHelixDB()` (exec `helix --version` + health check HTTP a localhost:6969), `checkDocker()` (exec `docker --version`), `checkGit()` (exec `git --version`). Cada uno retorna `*CheckResult` con Status y FixStep si aplica. Implementar `CheckAll()` (invoca todos en paralelo con goroutines) y `CheckOne()` (invoca uno específico).
- **Criterio:** `CheckAll()` retorna 5 resultados. `checkGo()` detecta versión de Go. `checkHelixDB()` detecta si HelixDB está corriendo vía `/health`. Tests con exec.Command mockeado.

#### T-0.7: Instalador de dependencias externas (install.go parte 1)
- **Archivos:** `internal/setup/install.go` (nuevo, ~90 líneas)
- **Depende de:** T-0.5
- **Descripción:** `Installer` struct con DryRun/Verbose/Platform. Método `InstallUV()`: ejecuta `curl -sSL https://astral.sh/uv/install.sh | bash`, idempotente (skip si `uv` ya en PATH). Método `InstallHelixDB()`: detecta OS/arch, descarga release de GitHub, extrae a `~/.local/bin/`, `chmod +x`, verifica con `helix --version`. Timeout 60s con 1 retry.
- **Criterio:** `InstallUV()` en dry-run no ejecuta nada. `InstallHelixDB()` descarga binario correcto para linux-amd64. Tests con mocks de exec.Command y http.Client.

#### T-0.8: Instalador de entorno local (install.go parte 2)
- **Archivos:** `internal/setup/install.go` (modificar, ~80 líneas)
- **Depende de:** T-0.7
- **Descripción:** Método `CreateVenv(toolsDir)`: ejecuta `uv venv` y `uv sync mcp-tools/pyproject.toml` en el directorio de tools. Método `BuildGoBinary(outputPath)`: ejecuta `go build -o <outputPath> ./cmd/zyrocli`. Método `RegisterMCPConfig()`: escribe configuración MCP en `~/.config/opencode/opencode.jsonc`. Método `GenerateConfig()`: genera struct `Config` y lo serializa a YAML.
- **Criterio:** `CreateVenv()` en dry-run muestra comandos sin ejecutar. `BuildGoBinary()` compila el binario. Tests con exec.Command mockeado.

#### T-0.9: Config struct + Load/Save (config.go)
- **Archivos:** `internal/setup/config.go` (nuevo, ~80 líneas)
- **Depende de:** T-0.5
- **Descripción:** `Config` struct con Version, Paths (HelixBinary, ZyroBinary, Venv, MCPTools, ConfigDir, SkillsDir, AuditDir), HelixDB (Host, Port, AutoStart), MCP (AutoRegister, LazyLoader). `PathsConfig`, `HelixConfig`, `MCPConfig` sub-structs. `DefaultConfig()` retorna configuración por defecto. `LoadConfig()` lee `~/.zyro/config.yaml`. `SaveConfig()` escribe. `ConfigPath()` retorna la ruta.
- **Criterio:** `DefaultConfig()` tiene valores sensatos. Ciclo Save→Load preserva datos. `ConfigPath()` apunta a `~/.zyro/config.yaml`. Tests con archivos temporales.

#### T-0.10: Doctor (diagnóstico y reparación)
- **Archivos:** `internal/setup/doctor.go` (nuevo, ~70 líneas)
- **Depende de:** T-0.7, T-0.9
- **Descripción:** `DoctorReport` struct con Timestamp/Status/Checks. `FixStep` struct con Description/Command/AutoFix. `Doctor` struct que wrappea Checker + Installer + Config. `NewDoctor(cfg)`. `Run(ctx, fix)` ejecuta checks y opcionalmente repara. `Fix(ctx)` repara problemas detectados. `Report(ctx)` genera reporte JSON-serializable.
- **Criterio:** `Run(ctx, false)` retorna reporte con checks. `Run(ctx, true)` ejecuta fixes. Tests con Checker mockeado.

#### T-0.11: Tests para check.go (verificadores)
- **Archivos:** `internal/setup/check_test.go` (nuevo, ~80 líneas)
- **Depende de:** T-0.6
- **Descripción:** Tests unitarios: `TestDetectOS` (linux/darwin), `TestDependencyChecker_CheckGo` (con exec.Command mock), `TestDependencyChecker_CheckGoMissing` (Go no instalado), `TestDependencyChecker_CheckUV`, `TestCheckAll`, `TestCheckOne`. Usar `execCmdFunc` reemplazable en tests.
- **Criterio:** `go test ./internal/setup/ -run TestCheck` pasa. Cobertura >80% en check.go.

#### T-0.12: Tests para install.go + config.go + doctor.go
- **Archivos:** `internal/setup/install_test.go`, `internal/setup/config_test.go`, `internal/setup/doctor_test.go` (nuevos, ~120 líneas total)
- **Depende de:** T-0.8, T-0.9, T-0.10
- **Descripción:** `TestInstaller_InstallUV` (idempotente con mock), `TestInstaller_InstallHelixDB` (descarga mockeada), `TestLoadConfig`, `TestDefaultConfig`, `TestSaveConfig`, `TestDoctor_Run`, `TestDoctor_Fix`, `TestDoctor_Report`.
- **Criterio:** `go test ./internal/setup/` pasa todos los tests. Cobertura >70%.

---

## Sprint 1: Agent-as-Validator
### Dependencias: Sprint 0

#### T-1.1: Modelos Pydantic (models.py)
- **Archivos:** `mcp-tools/models.py` (nuevo, ~80 líneas)
- **Depende de:** Nada (sprint 1)
- **Descripción:** Definir modelos: `Action` enum (create, update, search, skip), `HelixNodeOutput` (label, properties, project_id, requires_approval), `AgentDecision` (action, reasoning min_length=10, nodes list, requires_approval, metadata), `ZyroAgentInput` (protocol, version, request_id, phase, task, memory_context, boundari_phase, timeout_seconds, read_cap), `HelixSearchResult` (id, label, content, score, source), `HelixReadInput` (query, limit, node_labels). Todos con `model_config` para strict mode.
- **Criterio:** `AgentDecision(action="search", reasoning="x"*10)` valida. `AgentDecision(reasoning="short")` lanza ValidationError. `pytest mcp-tools/tests/ -k models` pasa.

#### T-1.2: Capacidades (capabilities.py)
- **Archivos:** `mcp-tools/capabilities.py` (nuevo, ~40 líneas)
- **Depende de:** T-1.1
- **Descripción:** `HelixReadCapability` dataclass con max_results=10, allowed_nodes tuple con labels permitidos. `AgentDependencies` dataclass con read_cap, phase, task_description, memory_context, boundari_phase, request_id.
- **Criterio:** `AgentDependencies(read_cap=HelixReadCapability(), phase="F0", task_description="test")` se construye sin errores.

#### T-1.3: Agente principal (agent.py parte 1 — setup)
- **Archivos:** `mcp-tools/agent.py` (nuevo, ~60 líneas)
- **Depende de:** T-1.1, T-1.2
- **Descripción:** Crear `zyro_agent: Agent[AgentDependencies, AgentDecision]` con `model="openai:gpt-5.2"`, `output_type=AgentDecision`, `deps_type=AgentDependencies`. System prompt con reglas: "NUNCA intentes escribir en HelixDB, NUNCA solicites tools de escritura, si detectas contradicción con memoria causal inclúyelo en reasoning". Función `run_agent(input_data, deps)` que ejecuta el agente.
- **Criterio:** `zyro_agent` es instancia de `Agent`. System prompt contiene las 3 reglas.

#### T-1.4: Tools read-only del agente (agent.py parte 2)
- **Archivos:** `mcp-tools/agent.py` (modificar, ~60 líneas)
- **Depende de:** T-1.3
- **Descripción:** Implementar 3 tools decoradas con `@zyro_agent.tool`: `search_code(ctx, input: HelixReadInput) -> list[HelixSearchResult]`, `search_skills(ctx, input: HelixReadInput) -> list[HelixSearchResult]`, `task_context(ctx, task_id: int) -> str`. Cada una usa `helix_client.py` como backend HTTP (solo lectura). Agregar tool `save_to_helix` con `requires_approval=True` que lanza `RuntimeError` (nunca se ejecuta desde el agente).
- **Criterio:** Las 4 tools aparecen en `zyro_agent._function_tools`. `save_to_helix` tiene `requires_approval=True`.

#### T-1.5: Approval Gate (approval.py)
- **Archivos:** `mcp-tools/approval.py` (nuevo, ~70 líneas)
- **Depende de:** T-1.1
- **Descripción:** `ApprovalGate` class con mode ("console" o "go_bridge"). `__init__(mode)`. `request_approval(decision, phase) -> bool` llama a `_console_approver` o `_go_bridge_approver`. `_console_approver`: imprime decisión en terminal, pide input sí/no. `_go_bridge_approver`: imprime JSON a stdout para que Go lo capture.
- **Criterio:** `ApprovalGate(mode="console").request_approval(decision, "F0")` imprime decisión y pide input. Tests con stdin mockeado.

#### T-1.6: Runner refactorizado (runner.py)
- **Archivos:** `mcp-tools/runner.py` (refactor completo, ~50 líneas)
- **Depende de:** T-1.3, T-1.4, T-1.5
- **Descripción:** Refactor completo de FastMCP server a orquestador async: `main()` lee `sys.stdin`, parsea JSON a `ZyroAgentInput`, construye `AgentDependencies`, llama a `run_agent()`, imprime `AgentDecision.model_dump_json()` por stdout. Reemplazar `server.run(transport="stdio")` por `asyncio.run(main())`. Eliminar dependencia de FastMCP.
- **Criterio:** `echo '{"protocol":"zyro-agent-v2","phase":"F0","task":"test"}' | python mcp-tools/runner.py` imprime JSON válido con `action` y `reasoning`.

#### T-1.7: Actualizar pyproject.toml + deprecar helix_write.py
- **Archivos:** `mcp-tools/pyproject.toml` (modificar), `mcp-tools/helix_write.py` (deprecar, ~10 líneas)
- **Depende de:** T-1.6
- **Descripción:** Agregar `pydantic-graph` a dependencias. Reemplazar `pydantic-ai` (sin version pin) por `pydantic-ai>=1.95`. Marcar `helix_write.py` con docstring `DEPRECATED: writes handled by Go SDK` y reducir a stub. Actualizar `py-modules` en `[tool.setuptools]`.
- **Criterio:** `uv sync --directory mcp-tools` instala sin errores.

#### T-1.8: Convertir tools existentes a formato agente
- **Archivos:** `mcp-tools/task_context.py` (modificar), `mcp-tools/search_code.py` (modificar), `mcp-tools/search_skills.py` (modificar), `mcp-tools/helix_client.py` (mantener)
- **Depende de:** T-1.4
- **Descripción:** Refactorizar cada tool existente para que sea llamable como función async independiente (no MCP tool). `task_context.py`: función `get_task_context(task_id) -> str` usando `helix_client.py`. `search_code.py`: función `search_code(query, limit) -> list[dict]`. `search_skills.py`: función `search_skills(query, limit) -> list[dict]`. `helix_client.py`: mantener como capa HTTP, asegurar que solo expone métodos GET/read.
- **Criterio:** `from task_context import get_task_context; get_task_context(1)` retorna string. `from search_code import search_code; search_code("test", 5)` retorna lista.

#### T-1.9: Tests Python (models + capabilities)
- **Archivos:** `mcp-tools/tests/test_models.py`, `mcp-tools/tests/test_capabilities.py` (nuevos, ~80 líneas)
- **Depende de:** T-1.1, T-1.2
- **Descripción:** Tests: `test_agent_decision_valid`, `test_agent_decision_invalid_reasoning`, `test_helix_node_output_valid`, `test_zyro_agent_input_valid`, `test_helix_read_capability_defaults`.
- **Criterio:** `pytest mcp-tools/tests/test_models.py test_models.py test_capabilities.py -v` pasa todos.

#### T-1.10: Tests Python (agent + approval + runner)
- **Archivos:** `mcp-tools/tests/test_agent.py`, `mcp-tools/tests/test_approval.py`, `mcp-tools/tests/test_runner.py` (nuevos, ~120 líneas)
- **Depende de:** T-1.5, T-1.6
- **Descripción:** Tests async: `test_run_agent_search` (mockea LLM), `test_agent_no_write_tools` (verifica que save_to_helix no se ejecuta), `test_approval_gate_console_approve` (monkeypatch stdin), `test_approval_gate_go_bridge`, `test_runner_stdin_json` (mockea sys.stdin). Usar `@pytest.mark.asyncio` y monkeypatch.
- **Criterio:** `pytest mcp-tools/tests/test_agent.py test_approval.py test_runner.py -v` pasa todos.

---

## Sprint 2: HelixDB SDK Go
### Dependencias: Sprint 0

#### T-2.1: Client Wrapper (helix.go)
- **Archivos:** `internal/db/helix/helix.go` (refactor, ~100 líneas)
- **Depende de:** T-0.3
- **Descripción:** Refactor completo: migrar de `buildV3Envelope` + `doQuery` raw HTTP a wrapper delgado sobre `github.com/helixdb/helix-db/sdks/go`. `ClientOptions` struct con BaseURL, Timeout (default 30s), MaxRetries (default 3). `Client` struct con `inner *helix.Client`. `NewClient(opts) (*Client, error)`. `Exec(ctx, q, out)` con retry loop (3x con backoff 100,200,300ms para ErrConnectionFailed). `HealthCheck(ctx)` (usa `helix.ReadQuery("health")`). `Close()`.
- **Criterio:** `go build ./internal/db/helix/` compila. `NewClient(ClientOptions{})` no retorna error. Tests con mock de helix.Client.

#### T-2.2: Tipos y errores actualizados (types.go + errors.go)
- **Archivos:** `internal/db/helix/types.go` (modificar), `internal/db/helix/errors.go` (modificar, ~50 líneas total)
- **Depende de:** T-2.1
- **Descripción:** `types.go`: reemplazar tipos propios por wrappers de tipos del SDK oficial. Mantener compatibilidad con código existente. `errors.go`: reemplazar `ErrNotFound`, `ErrConnectionFailed`, `ErrConflict` por variables que wrappean `helix.HelixError`. Agregar `IsHelixNotFound(err)`, `IsHelixConflict(err)` helpers.
- **Criterio:** `errors.Is(err, ErrConnectionFailed)` funciona correctamente. Tests verifican type assertions.

#### T-2.3: Row types para queries (queries.go parte 1)
- **Archivos:** `internal/db/helix/queries.go` (nuevo, ~60 líneas)
- **Depende de:** T-2.1
- **Descripción:** Definir Row types para resultados de queries: `TaskRow` (ID, Name, Description, Phase, Status, CreatedAt), `CodeNodeRow` (ID, Path, Summary, Language, Hash), `FactRow` (ID, Type, Content, Salience, Confidence, Phase, IsActive), `ProjectRow` (ID, Name, Description, Status, CurrentPhase), `SkillRow` (ID, Name, Type, Version), `PatternRow` (ID, Name, Description, Language). Todos con tags JSON.
- **Criterio:** `TaskRow{}` tiene campos correctos con tags JSON. `json.Marshal(TaskRow{ID: 1})` produce `{"$id":1,...}`.

#### T-2.4: Query builders parte 1 (queries.go parte 2)
- **Archivos:** `internal/db/helix/queries.go` (modificar, ~80 líneas)
- **Depende de:** T-2.3
- **Descripción:** Implementar query builders que retornan `helix.Request`: `FindTask(name, projectID)`, `UpsertCodeNode(projectID, path, summary, language, hash)` (con upsert condicional usando `VarAsIf`), `FindProject(name)`, `ListFactsByPhase(phase, tenantID, limit)`. Cada builder usa `helix.ReadQuery/WriteQuery`, `ParamString`, `G().NWithLabel()`, `Where`, `ValueMap`.
- **Criterio:** `FindTask("auth", 1005)` retorna `helix.Request` válido. Tests verifican que la request tiene los steps correctos.

#### T-2.5: Query builders parte 2 (queries.go parte 3)
- **Archivos:** `internal/db/helix/queries.go` (modificar, ~80 líneas)
- **Depende de:** T-2.4
- **Descripción:** Implementar: `CreateFact(label, props, embedding)` (crea nodo Fact con embedding), `CreateEdge(fromID, toID, edgeType, props)`, `FindSkills(query, limit)`, `FindPatterns(query, language, limit)`. Usar `AddN`, `AddE`, `TextSearchNodes` del SDK oficial.
- **Criterio:** `CreateFact("Fact", {"content":"test"}, []float32{0.1,0.2})` genera request con AddN y embedding. Tests verifican estructura.

#### T-2.6: Búsqueda híbrida (search.go parte 1)
- **Archivos:** `internal/db/helix/search.go` (nuevo, ~90 líneas)
- **Depende de:** T-2.1, T-2.3
- **Descripción:** `SearchResult` struct con ID, Label, Content, Score, Source. `HybridSearchOptions` con MaxResults (10), RRFFusionK (60), NodeLabels, TenantID, MinScore. `HybridSearch(ctx, client, query, embedding, opts)` ejecuta `vectorSearch` y `textBM25Search` en paralelo con `sync.WaitGroup` + channel. `vectorSearch()`: usa `VectorSearchNodes` del SDK. `textBM25Search()`: usa `TextSearchNodes` del SDK.
- **Criterio:** `HybridSearch` con embedding vacío cae a solo BM25. Tests con mocks de cliente verifican paralelismo.

#### T-2.7: RRF Fusion (search.go parte 2)
- **Archivos:** `internal/db/helix/search.go` (modificar, ~50 líneas)
- **Depende de:** T-2.6
- **Descripción:** `fuseRRF(vector, text []SearchResult, k, maxResults)`: implementa Reciprocal Rank Fusion. `RRFScore(d) = sum(1 / (k + rank_i(d)))`. Normalizar scores a [0,1] por lista. Mergear resultados, ordenar por score descendente, deduplicar por ID, retornar top-N. Manejar casos: vector o text vacío.
- **Criterio:** fuseRRF con 3 resultados vector y 2 text produce ranking fusionado. IDs duplicados aparecen una sola vez. fuseRRF empty retorna vacío. Tests parametrizados.

#### T-2.8: Traversals (traverse.go parte 1)
- **Archivos:** `internal/db/helix/traverse.go` (nuevo, ~70 líneas)
- **Depende de:** T-2.1, T-2.3
- **Descripción:** `LibraryInfo` struct. `FactWithPath` struct con ID, Type, Content, Phase, Confidence. `ProjectContext` struct con Project, Tasks, Patterns, Skills, Libraries. `ContradictionPair` struct con FactA, FactB, Similarity. `DiscoverCrossProjectSkills(ctx, client, skillName)`: Skill ← REQUIRES_SKILL ← Project → USES_LIB → Library usando In/Out/Dedup. `TraverseProjectContext(ctx, client, projectID)`: arma contexto completo de un proyecto.
- **Criterio:** `DiscoverCrossProjectSkills` retorna `[]LibraryInfo`. Tests con mock de Exec verifican steps de traversal.

#### T-2.9: Traversals causales (traverse.go parte 2)
- **Archivos:** `internal/db/helix/traverse.go` (modificar, ~50 líneas)
- **Depende de:** T-2.8
- **Descripción:** `TraverseCausalChain(ctx, client, factID, maxDepth)`: navega cadena causal con `Repeat(Out("CAUSED","PRECEDES","DERIVES_FROM").MaxDepth(maxDepth).Dedup())`. `FindContradictions(ctx, client, tenantID, threshold)`: busca pares de Facts CONTRADICTS con filter por similaridad de embedding.
- **Criterio:** `TraverseCausalChain(ctx, client, 1, 3)` genera request con Repeat. Tests con mock.

#### T-2.10: Pipeline de embeddings (embedding.go)
- **Archivos:** `internal/db/helix/embedding.go` (nuevo, ~100 líneas)
- **Depende de:** T-2.1
- **Descripción:** `EmbeddingProvider` enum (ProviderOpenAI, ProviderOllama). `EmbeddingConfig` con Provider, Model, Dims, APIKey, BaseURL, BatchSize (20), CacheSize (1000), MaxRetries (3). `EmbeddingService` struct con config, http.Client, lru.Cache, mutex. `NewEmbeddingService(config)`. `Embed(ctx, text)` (check cache → API → store cache). `EmbedBatch(ctx, texts)` (procesa en batches). `embedOpenAI(ctx, texts)` (llama API OpenAI). `embedOllama(ctx, texts)` (llama API Ollama local). Fallback: si OpenAI falla, reintenta con Ollama.
- **Criterio:** `Embed(ctx, "test")` retorna `[]float32` de 1536 dims (OpenAI) o 768 dims (Ollama). Cache hits no llaman API. Tests con http.Client mock.

#### T-2.11: Gestión de índices (indexes.go)
- **Archivos:** `internal/db/helix/indexes.go` (nuevo, ~50 líneas)
- **Depende de:** T-2.1
- **Descripción:** `IndexType` enum (IndexVector, IndexText, IndexEquality, IndexRange, IndexUnique). `IndexSpec` struct con Label, Property, IndexType, TenantProperty. `EnsureIndexes(ctx, client, specs)`: itera specs, llama `CreateIndexIfNotExists` por cada uno. `DefaultIndexes()`: retorna indices para todos los labels del schema (Developer, Project, CodeNode, Skill, Task, Pattern, Library, Fact). `CreateIndexIfNotExists(spec)`: genera request con `CreateIndexIfNotExists`.
- **Criterio:** `DefaultIndexes()` retorna ≥8 specs. Tests verifican que cada spec tiene Label y Property no vacíos.

#### T-2.12: Tests para HelixDB SDK
- **Archivos:** `internal/db/helix/helix_test.go` (refactor, ~150 líneas)
- **Depende de:** T-2.1 a T-2.11
- **Descripción:** Tests unitarios: `TestNewClient`, `TestClientExec`, `TestClientExecRetry` (con mock de conexión fallida), `TestFindTaskQuery`, `TestUpsertCodeNodeQuery`, `TestCreateFactQuery`, `TestHybridSearch`, `TestFuseRRF`, `TestFuseRRFEmpty`, `TestDiscoverCrossProjectSkills`, `TestTraverseCausalChain`, `TestEmbeddingService`, `TestEmbeddingServiceCache`, `TestEmbeddingServiceFallback`, `TestEnsureIndexes`, `TestDefaultIndexes`. Mocks: helix.Client interface, http.Client.
- **Criterio:** `go test ./internal/db/helix/ -v` pasa todos. Cobertura >75%.

---

## Sprint 3: Boundari por Fase
### Dependencias: Sprint 1

#### T-3.1: Políticas YAML (5 archivos)
- **Archivos:** 
  - `boundari/phase0-boundari.yaml` (nuevo, ~25 líneas)
  - `boundari/phase1-boundari.yaml` (nuevo, ~30 líneas)
  - `boundari/phase2-boundari.yaml` (nuevo, ~30 líneas)
  - `boundari/phase3-boundari.yaml` (nuevo, ~30 líneas)
  - `boundari/phase4-boundari.yaml` (nuevo, ~25 líneas)
- **Depende de:** Nada (sprint 3)
- **Descripción:** Crear 5 archivos YAML con políticas por fase. Cada uno con: version, phase, description, budget (max_tool_calls, max_runtime_seconds, max_cost_usd), tools (map de tool_name → {allow, deny, approval}). F0: solo lectura. F1: lectura + web_fetch con approval condicional. F2: escritura planos con approval. F3: implementación intensiva con approval condicional. F4: solo lectura + approval para correcciones. Seguir el diseño exacto de la sección 5.6 del design.
- **Criterio:** Cada YAML se parsea correctamente con `yaml.Unmarshal`. F0 no permite write_file. F3 permite write_file en src/ sin approval. `python3 -c "import yaml; yaml.safe_load(open('boundari/phase0-boundari.yaml'))"` funciona.

#### T-3.2: Tipos Boundari (types.go)
- **Archivos:** `internal/boundari/types.go` (nuevo, ~70 líneas)
- **Depende de:** T-3.1
- **Descripción:** `Phase` enum (PhaseF0-F4). `Policy` struct con Version, Phase, Description, Budget, Tools (map[string]ToolPolicy), Data, Outputs, Tests. `Budget` struct con MaxToolCalls, MaxRuntimeSecs, MaxCostUSD, MaxTokens. `ToolPolicy` struct con Allow, Deny, Approval (*ApprovalPolicy), Scopes, Risk. `ApprovalPolicy` struct con Required, When (condición string). `EnforcementResult` struct con Allowed, ToolName, Reason, RequiresApproval. `AuditEvent` struct con Timestamp, Phase, ToolName, Args, Allowed, Reason, Duration. `BudgetUsage` struct con ToolCalls, RuntimeSecs, CostUSD, Tokens. Todos con tags YAML y JSON.
- **Criterio:** `Policy{}` se serializa/deserializa con YAML. `EnforcementResult{Allowed: true}` tiene JSON correcto.

#### T-3.3: Loader de políticas (loader.go)
- **Archivos:** `internal/boundari/loader.go` (nuevo, ~60 líneas)
- **Depende de:** T-3.2
- **Descripción:** `LoadPolicy(phase, searchDirs...)` busca archivo `phase{N}-boundari.yaml` en searchDirs, lo parsea con `yaml.Unmarshal`, retorna `*Policy`. `LoadDefaultPolicy(phase)`: retorna política hardcodeada como fallback si no encuentra archivo. `ValidatePolicy(p)`: verifica que todos los campos requeridos estén presentes, que budget tenga valores positivos, que tools tenga entradas.
- **Criterio:** `LoadPolicy("F0", ["boundari/"])` carga fase0-boundari.yaml. `LoadPolicy("F9")` retorna default. `ValidatePolicy` detecta budget negativo.

#### T-3.4: Enforcer Go (enforcer.go parte 1)
- **Archivos:** `internal/boundari/enforcer.go` (nuevo, ~80 líneas)
- **Depende de:** T-3.2, T-3.3
- **Descripción:** `Enforcer` struct con policy, usage (BudgetUsage), startAt. `NewEnforcer(policy)`. Método `CheckTool(toolName, args) -> EnforcementResult`: 1) Budget check (tool_calls, runtime), 2) Deny check (prioridad sobre allow), 3) Allow check, 4) Approval condition evaluation (safe_eval de condición When). `IsBudgetExceeded() -> bool`. `Usage() -> BudgetUsage`.
- **Criterio:** `CheckTool("write_file", nil)` con policy F0 retorna `{Allowed: false, Reason: "denied"}`. `CheckTool("read_file", nil)` con F0 retorna `{Allowed: true}`. `IsBudgetExceeded()` true después de exceder maxToolCalls.

#### T-3.5: Auditoría Enforcer (enforcer.go parte 2)
- **Archivos:** `internal/boundari/enforcer.go` (modificar, ~50 líneas)
- **Depende de:** T-3.4
- **Descripción:** `LogAudit(event AuditEvent)`: agrega evento a log interno. `SaveAuditLog(ctx, path)`: escribe log en formato JSONL (una línea por evento) en `~/.zyro/audit/<phase>-<timestamp>.jsonl`. `Reset()`: reinicia usage y startAt. Método `checkApprovalCondition(condition, args) -> bool`: evalúa condición safe usando `strings.Contains` y comparaciones simples (sin exec).
- **Criterio:** `SaveAuditLog` produce archivo JSONL válido. Cada línea es JSON parseable con `AuditEvent`. `Reset()` pone ToolCalls a 0.

#### T-3.6: Wrapper Python (boundari_wrapper.py)
- **Archivos:** `mcp-tools/boundari_wrapper.py` (nuevo, ~80 líneas)
- **Depende de:** T-3.1
- **Descripción:** `BoundariWrapper` class: `__init__(phase, policies_dir, audit_dir)`. Carga política YAML. `_load_policy()`: busca archivo `<phase>-boundari.yaml` en policies_dir. `wrap_tool(tool_name, tool_func, raise_on_denied)`: retorna función wrapper async que: (1) checkea budget, (2) checkea política (allow/deny/approval), (3) si deny → raise o return, (4) si approval required → log + raise ApprovalRequired, (5) ejecuta tool, (6) registra auditoría. `_check_policy_fallback(tool_name, args) -> Decision`: implementación hardcodeada de política por fase (fallback si no hay YAML). `save_audit(phase)` -> str.
- **Criterio:** `BoundariWrapper(phase="F0").wrap_tool("write_file", lambda: None)({"path":"test"})` lanza PermissionError. `BoundariWrapper(phase="F3").wrap_tool("write_file", lambda x: x)({"path":"src/app.ts"})` ejecuta correctamente.

#### T-3.7: Integrar Boundari en agent.py
- **Archivos:** `mcp-tools/agent.py` (modificar, +20 líneas)
- **Depende de:** T-3.6
- **Descripción:** En `run_agent()`, antes de ejecutar el agente: (1) crear `BoundariWrapper(phase=input_data.boundari_phase)`, (2) envolver cada tool registrada con `wrapper.wrap_tool()`. Si Boundari falla (import error), usar fallback hardcodeado. Agregar logging de auditoría post-ejecución.
- **Criterio:** `run_agent(input_data, deps)` con `boundari_phase="F0"` envuelve tools. Intento de write desde F0 lanza error capturado.

#### T-3.8: Tests para Boundari
- **Archivos:** 
  - `internal/boundari/boundari_test.go` (nuevo, ~120 líneas)
  - `mcp-tools/tests/test_boundari_wrapper.py` (nuevo, ~60 líneas)
- **Depende de:** T-3.1 a T-3.7
- **Descripción:** Tests Go: `TestLoadPolicy`, `TestLoadPolicyNotFound`, `TestLoadDefaultPolicy`, `TestValidatePolicy`, `TestEnforcer_CheckTool_Allow`, `TestEnforcer_CheckTool_Deny`, `TestEnforcer_CheckTool_NotFound`, `TestEnforcer_CheckTool_BudgetExceeded`, `TestEnforcer_CheckTool_ApprovalRequired`, `TestEnforcer_SaveAuditLog`, `TestAllPoliciesLoad` (carga los 5 YAML), `TestPhase0NoWriteTools`, `TestPhase3WriteFileConditional`. Tests Python: `test_boundari_wrapper_f0_blocks_write`, `test_boundari_wrapper_f3_conditional`, `test_boundari_wrapper_audit_log`.
- **Criterio:** `go test ./internal/boundari/ -v` pasa todos. `pytest mcp-tools/tests/test_boundari_wrapper.py -v` pasa todos.

---

## Sprint 4: Memoria Causal
### Dependencias: Sprint 2

#### T-4.1: Schema de memoria (schema.go)
- **Archivos:** `internal/memory/schema.go` (nuevo, ~80 líneas)
- **Depende de:** T-2.3
- **Descripción:** `FactType` string enum: FactDecision, FactError, FactPreference, FactPattern, FactDependency, FactObservation. `CausalEdgeType` string enum: EdgeCaused, EdgePrecedes, EdgeContradicts, EdgeSupports, EdgeRequires, EdgeDerivesFrom, EdgeReferences (7 aristas). `Fact` struct completo con todos los campos del diseño (ID, Type, Content, Embedding, Salience, Confidence, Source, Phase, CreatedAt, LastAccessedAt, AccessCount, DecayRate, ExpiresAt, IsActive, IsStale, ProjectID, Metadata). `CausalEdge` struct con ID, FromID, ToID, Type, CreatedAt, Properties. `ContradictionPair`, `RecallOpts`, `MemoryResult`, `DecayConfig`, `ContradictionStrategy` types.
- **Criterio:** `Fact{Type: FactDecision, Content: "test"}` se construye. `json.Marshal` produce campos correctos. Tests de tipos.

#### T-4.2: Interfaz EngramStore (memory.go)
- **Archivos:** `internal/memory/memory.go` (nuevo, ~50 líneas)
- **Depende de:** T-4.1
- **Descripción:** `EngramStore` interface con métodos: `SaveFact(ctx, fact) (int64, error)`, `SaveFactsBatch(ctx, facts) ([]int64, error)`, `AddCausalEdge(ctx, edge) error`, `RecallMemories(ctx, opts) ([]*MemoryResult, error)`, `DetectContradictions(ctx, projectID, threshold) ([]ContradictionPair, error)`, `ResolveContradiction(ctx, pair, strategy) error`, `ReinforceSalience(ctx, factIDs) error`, `DecayAndRefresh(ctx, projectID) error`, `GetFactByID(ctx, factID) (*Fact, error)`, `GetCausalChain(ctx, factID, maxDepth) ([]*Fact, error)`. Documentar cada método con comentarios Go.
- **Criterio:** Todos los métodos de la interfaz están documentados. `EngramStore` se puede mockear para tests.

#### T-4.3: Store implementation SaveFact (store.go parte 1)
- **Archivos:** `internal/memory/store.go` (nuevo, ~70 líneas)
- **Depende de:** T-4.1, T-4.2, T-2.10
- **Descripción:** `HelixEngramStore` struct con client, embeddingSvc, defaultDecayRate. `NewHelixEngramStore(client, embeddingSvc)`. `SaveFact(ctx, fact)`: si embedding está vacío, computar con `embeddingSvc.Embed()`, crear nodo Fact en HelixDB usando `db.CreateFact()`. `SaveFactsBatch(ctx, facts)`: itera facts, computa embeddings en batch con `EmbedBatch()`, guarda todos. `AddCausalEdge(ctx, edge)`: crea edge entre dos Facts usando `db.CreateEdge()`.
- **Criterio:** `SaveFact` guarda fact con embedding. `SaveFactsBatch` procesa 10 facts en 1 batch de embeddings. Tests con mocks.

#### T-4.4: Store batch operations (store.go parte 2)
- **Archivos:** `internal/memory/store.go` (modificar, ~50 líneas)
- **Depende de:** T-4.3
- **Descripción:** `GetFactByID(ctx, factID)`: busca fact por ID usando query tipada. `ReinforceSalience(ctx, factIDs)`: actualiza salience, access_count, last_accessed_at para facts específicos usando fórmula: `salience += 0.3 * (1 - salience)`. `deactivateFact(ctx, factID, reason)`: marca fact como IsActive=false, agrega metadata con razón.
- **Criterio:** `GetFactByID(1)` retorna Fact. `ReinforceSalience` incrementa salience correctamente. Tests.

#### T-4.5: RecallMemories (recall.go parte 1)
- **Archivos:** `internal/memory/recall.go` (nuevo, ~70 líneas)
- **Depende de:** T-4.2, T-2.6
- **Descripción:** `RecallMemories(ctx, opts)`: construye query híbrida con: (1) vector search sobre embedding del query_text, (2) BM25 search sobre content, (3) RRF fusion con k=60. Filtros: MinSalience (default 0.2), FactTypes, IncludeStale (default false), Phase, ProjectID. Limita a MaxResults. Retorna `[]*MemoryResult` con Score calculado.
- **Criterio:** `RecallMemories(ctx, RecallOpts{QueryText: "test", MaxResults: 5})` retorna resultados ordenados por score. Tests con HybridSearch mockeado.

#### T-4.6: GetCausalChain (recall.go parte 2)
- **Archivos:** `internal/memory/recall.go` (modificar, ~40 líneas)
- **Depende de:** T-4.5, T-2.9
- **Descripción:** `GetCausalChain(ctx, factID, maxDepth)`: navega grafo causal usando `db.TraverseCausalChain` (Repeat con Out de CAUSED, PRECEDES, DERIVES_FROM). Retorna `[]*Fact` ordenados por profundidad. Incluye el fact de origen. Limita a maxDepth (default 5).
- **Criterio:** `GetCausalChain(ctx, 1, 3)` retorna facts en orden. Tests con TraverseCausalChain mockeado.

#### T-4.7: Detección de contradicciones (contradictions.go)
- **Archivos:** `internal/memory/contradictions.go` (nuevo, ~60 líneas)
- **Depende de:** T-4.2
- **Descripción:** `DetectContradictions(ctx, projectID, threshold)`: (1) busca todos los Facts activos del proyecto, (2) compara embeddings por pares usando cosine similarity, (3) si similarity > threshold (default 0.85) y tipos son opuestos (decision vs decision, preference vs preference), marca como par de contradicción. `ResolveContradiction(ctx, pair, strategy)`: aplica estrategia (NewestWins, HighestConfidence, KeepBoth). Si no es KeepBoth, marca fact perdedor como IsActive=false.
- **Criterio:** `DetectContradictions` con 2 facts similares detecta par. `ResolveContradiction(NewestWins)` desactiva fact más viejo. Tests con Facts mockeados.

#### T-4.8: Decaimiento Ebbinghaus (decay.go)
- **Archivos:** `internal/memory/decay.go` (nuevo, ~50 líneas)
- **Depende de:** T-4.2
- **Descripción:** `DecayAndRefresh(ctx, projectID)`: (1) carga todos los Facts activos del proyecto, (2) para cada fact calcula `newSalience = salience * e^(-decayRate * daysSinceAccess)`, (3) si `newSalience < threshold` (0.15) marca IsStale=true, (4) si `expiresAt < now` marca IsActive=false. `ReinforceSalience(ctx, factIDs)`: `salience += 0.3 * (1 - salience)` para cada fact, incrementa access_count, actualiza last_accessed_at.
- **Criterio:** Fact con salience 0.7, decay 0.05, 30 días sin acceso: salience ≈ 0.156. ReinforceSalience: 0.5 → 0.65. Tests verifican fórmula de Ebbinghaus.

#### T-4.9: Memory pre/post phase hooks (memory_hook.go)
- **Archivos:** `internal/scheduler/memory_hook.go` (nuevo, ~80 líneas)
- **Depende de:** T-4.5, T-4.3, T-4.7, T-4.8
- **Descripción:** `MemoryHooks` struct con store, embeddingSvc, factExtractorPath. `PrePhase(ctx, phase, taskDesc)`: llama `RecallMemories`, formatea como string "MEMORIA CAUSAL:" con bullets para cada fact (Type, Content, Confidence). `PostPhase(ctx, phase, conversationLog)`: (1) guarda log temporal, (2) ejecuta `python fact_extractor.py --input <log> --phase <phase>`, (3) parsea output JSON, (4) computa embeddings, (5) guarda facts con SaveFactsBatch, (6) crea edges causales entre facts de misma fase, (7) ejecuta DetectContradictions + ResolveContradiction (NewestWins), (8) ejecuta ReinforceSalience. `formatMemoryForPrompt(facts) -> string`: formatea facts como contexto inyectable.
- **Criterio:** `PrePhase` con 0 facts retorna string vacío. `PrePhase` con 3 facts produce "MEMORIA CAUSAL:\n• decision: ...". `PostPhase` ejecuta extractor y guarda facts.

#### T-4.10: Extractor de hechos Python (fact_extractor.py)
- **Archivos:** `agents/fact_extractor.py` (nuevo, ~80 líneas)
- **Depende de:** T-4.1
- **Descripción:** Script CLI: `python fact_extractor.py --input <log.json> --phase F1`. `FACT_PATTERNS` dict con 6 tipos: decision (regex: "vamos a usar|decidimos"), error ("error|bug|fallo"), preference ("prefiero|mejor usar"), pattern ("patrón|arquitectura"), dependency ("dependemos de|requiere"), observation ("observo|noto|detecto"). `extract_facts(log_text, phase)` -> list[dict] con type, content, salience=0.7, confidence=0.8, source="extractor:llm", phase, decay_rate=0.05. `extract_facts_llm(log_text, phase, model)` -> list[dict] (usando Ollama). `main()`: parsea args, lee JSON de input, extrae facts, imprime JSON.
- **Criterio:** `extract_facts("vamos a usar Go SDK oficial", "F0")` retorna fact type=decision. Output JSON tiene `{"facts": [...]}`. `python agents/fact_extractor.py --input <(echo '{"conversation":"decidimos usar SQLC"}') --phase F1` imprime JSON válido.

#### T-4.11: Integrar hooks en scheduler (scheduler.go + phase.go)
- **Archivos:** `internal/scheduler/scheduler.go` (modificar, +15 líneas), `internal/scheduler/phase.go` (modificar, +10 líneas)
- **Depende de:** T-4.9
- **Descripción:** `scheduler.go`: en `RunPhase`, antes de ejecutar la fase llamar `hooks.PrePhase()` e inyectar memoria en el contexto (agregar a `Config`), después de ejecutar llamar `hooks.PostPhase()` con el log de la fase. `phase.go`: agregar campo `MemoryContext string` a `Result` struct, agregar campo `MemoryHooks` a `Config`.
- **Criterio:** `RunPhase` con hooks activos inyecta memoria antes de ejecutar. Result contiene memory_context. Tests con MemoryHooks mockeado.

#### T-4.12: Tests para store + recall + hooks
- **Archivos:** `internal/memory/memory_test.go` (nuevo, ~150 líneas)
- **Depende de:** T-4.3 a T-4.11
- **Descripción:** Tests: `TestFactTypes`, `TestCausalEdgeTypes`, `TestHelixEngramStore_SaveFact` (con mock de embeddingSvc), `TestHelixEngramStore_SaveFactsBatch`, `TestHelixEngramStore_AddCausalEdge`, `TestHelixEngramStore_RecallMemories`, `TestHelixEngramStore_RecallMemories_Empty`, `TestHelixEngramStore_GetCausalChain`, `TestDetectContradictions`, `TestResolveContradiction_NewestWins`, `TestResolveContradiction_HighestConfidence`, `TestDecayAndRefresh_Ebbinghaus`, `TestReinforceSalience`, `TestFormatMemoryForPrompt`, `TestPrePhase`, `TestPostPhase`. Mocks: embed.EmbeddingService, db.Client.
- **Criterio:** `go test ./internal/memory/ -v` pasa todos. Cobertura >70%.

#### T-4.13: Tests Python para fact_extractor
- **Archivos:** `agents/tests/test_fact_extractor.py` (nuevo, ~60 líneas)
- **Depende de:** T-4.10
- **Descripción:** Tests: `test_extract_facts_decision`, `test_extract_facts_error`, `test_extract_facts_preference`, `test_extract_facts_no_match` (texto sin patrones retorna lista vacía), `test_extract_facts_short_content` (menos de 10 chars se filtra), `test_main_cli` (mockea sys.argv y open).
- **Criterio:** `pytest agents/tests/test_fact_extractor.py -v` pasa todos.

#### T-4.14: Embedding Harness MCP (embedding_harness.py)
- **Archivos:** `mcp-tools/embedding_harness.py` (nuevo, ~120 líneas)
- **Depende de:** T-4.3
- **Descripción:** MCP server separado implementado con FastMCP. Tools: `embed(text) → vector`, `embed_batch(texts) → vectors`, `status() → dict`. Cache LRU en disco (`~/.zyro/embedding-cache/`). Pipeline de prioridad: Ollama (mxbai-embed-large) → Scaleway (qwen3-embedding-8b) → GitHub Models/Cohere → BM25 fallback. `_get_embedding(text)`: intenta proveedores en orden, retorna nil si todos fallan. Cache key = SHA256 del texto.
- **Criterio:** `embed("test")` retorna `list[float]` con 768 dims (o 0 si no hay proveedor). `status()` reporta `provider`, `model`, `cache_size`, `available`.

#### T-4.15: Instalación interactiva de embeddings en zyro setup
- **Archivos:** `internal/setup/embedding.go` (nuevo, ~80 líneas)
- **Depende de:** T-0.1
- **Descripción:** Agregar paso interactivo en `zyro setup`: (1) preguntar si instalar Ollama, (2) detectar GPU con `nvidia-smi` / `rocminfo`, (3) instalar Ollama + pull `mxbai-embed-large`, (4) preguntar si configurar fallback API (Scaleway/GitHub Models/Cohere), (5) guardar config en `~/.zyro/config.yaml`. `detectGPU() → string`: retorna "nvidia", "amd", o "cpu". `installOllama(gpuType)`: ejecuta script de instalación según GPU. `pullModel(model)`: ejecuta `ollama pull`. `setupEmbeddingFallback() → EmbeddingConfig`: menú interactivo de APIs.
- **Criterio:** `zyro setup` pregunta interactivamente sobre embeddings. Detecta GPU correctamente (nvidia-smi/rocminfo). Configura embeddings en `~/.zyro/config.yaml`.

---

## Sprint 5: OpenCode + Boomerang
### Dependencias: Sprint 3 + Sprint 4

#### T-5.1: Orquestador Boomerang — tipos y constructor (orchestrator.go parte 1)
- **Archivos:** `internal/boomerang/orchestrator.go` (nuevo, ~60 líneas)
- **Depende de:** T-4.2, T-3.3
- **Descripción:** `PhaseConfig` struct con Phase, TaskDesc, ProjectID, MemoryLimit, Iterations, Timeout. `PhaseResult` struct con Phase, Success, Iterations, MemoryUsed, TasksPlanned, NodesCreated, GitStatus, QualityOK, FactsSaved, Duration (time.Duration en JSON como ms), Error. `BoomerangOrchestrator` struct con memoryStore (EngramStore), boundariLoader, delegateSvc, gitChecker, qualityGate, saveService, maxIterations (default 3). `NewBoomerangOrchestrator(store, bl)`.
- **Criterio:** `NewBoomerangOrchestrator(mockStore, mockLoader)` se construye. `PhaseConfig{}` tiene defaults correctos.

#### T-5.2: Ciclo Boomerang RunPhase (orchestrator.go parte 2)
- **Archivos:** `internal/boomerang/orchestrator.go` (modificar, ~100 líneas)
- **Depende de:** T-5.1
- **Descripción:** `RunPhase(ctx, config) -> (*PhaseResult, error)`: ciclo completo de 6 pasos: (1) MemoryStep → consulta memoria causal, (2) ThinkStep → planifica DAG de tareas según fase, (3) DelegateStep → reparte tareas a subagentes OpenCode (paralelo por grupos), (4) GitStep → verifica estado del repo, (5) QualityStep → valida resultados con loop de retry (max 3 iteraciones, si fail → redelegate), (6) SaveStep → guarda decisiones y hechos. Mide duración total. Retorna PhaseResult con todos los campos poblados.
- **Criterio:** `RunPhase` ejecuta los 6 pasos en orden. Si Quality falla, redelegate (max 3). PhaseResult tiene Success=true si todo OK. Tests con todos los pasos mockeados.

#### T-5.3: Memory Step (memory.go)
- **Archivos:** `internal/boomerang/memory.go` (nuevo, ~40 líneas)
- **Depende de:** T-5.2, T-4.5
- **Descripción:** `func (o *BoomerangOrchestrator) MemoryStep(ctx, phase, taskDesc) (string, error)`: llama `store.RecallMemories` con opts de fase actual, filtra por MinSalience=0.2, formatea como string de contexto usando `formatMemoryForPrompt`. Si no hay memoria, retorna string vacío. Loggea cantidad de hechos recuperados.
- **Criterio:** `MemoryStep` con store mock que retorna facts produce string con formato. Con 0 facts retorna "". Tests.

#### T-5.4: Think Step (think.go)
- **Archivos:** `internal/boomerang/think.go` (nuevo, ~70 líneas)
- **Depende de:** T-5.2
- **Descripción:** `TaskDAG` struct con Tasks ([]TaskSpec), Deps ([][2]int), ParallelGroups ([][]int). `TaskSpec` struct con ID, Name, Description, Agent, Tags, DependsOn. `ThinkStep(ctx, phase, memoryContext) -> *TaskDAG`: genera DAG de tareas según la fase actual. `generateDAGForPhase(phase, memoryContext) -> *TaskDAG`: F0: patrones, librerías, skills en paralelo (3 grupos). F1: investigación web + búsqueda. F2: planos. F3: implementación. F4: revisión. Cada fase tiene tareas específicas.
- **Criterio:** `ThinkStep` con phase F0 retorna DAG con 3 tareas en paralelo. F3 retorna DAG con tareas de implementación. Tests.

#### T-5.5: Delegate Step (delegate.go)
- **Archivos:** `internal/boomerang/delegate.go` (nuevo, ~80 líneas)
- **Depende de:** T-5.2, T-3.4
- **Descripción:** `DelegateService` struct con opencodeBin (path a binario OpenCode). `NewDelegateService(opencodeBin)`. `DelegateStep(ctx, dag, phase, boundariPolicy) -> *DelegateResult`: (1) para cada grupo paralelo en dag.ParallelGroups, lanza subprocesos OpenCode en paralelo (uno por tarea), (2) cada subproceso ejecuta `opencode subagent <agent> --param task=<task>`, (3) espera a todos los grupos secuencialmente (grupos paralelos internamente), (4) recolecta resultados. `DelegateResult` con NodesCreated, TaskResults (map[string]TaskResult). `TaskResult` con TaskName, Success, Output, Nodes.
- **Criterio:** `DelegateStep` con DAG de 3 tareas paralelas lanza 3 subprocesos. `DelegateStep` con grupos secuenciales espera grupo 1 antes de grupo 2. Tests con exec.Command mockeado.

#### T-5.6: Git Step (git.go)
- **Archivos:** `internal/boomerang/git.go` (nuevo, ~50 líneas)
- **Depende de:** T-5.2
- **Descripción:** `GitChecker` struct (sin estado). `NewGitChecker()`. `Status(ctx) -> *GitStatus`: ejecuta `git status --porcelain`, parsea output. `GitStatus` struct con Clean (bool), Branch, Changed (int), Untracked (int), Ahead, Behind, Error. `DiffCount(ctx) -> int`: cuenta archivos modificados. `GitStep(ctx) -> *GitStatus`: wrappea Status, loggea resultado.
- **Criterio:** `GitChecker().Status(ctx)` en repo limpio retorna Clean=true. En repo sucio retorna Changed>0. Tests con git exec mockeado.

#### T-5.7: Quality Step (quality.go)
- **Archivos:** `internal/boomerang/quality.go` (nuevo, ~60 líneas)
- **Depende de:** T-5.2
- **Descripción:** `QualityGate` struct. `QualityResult` struct con Passed (bool), Issues ([]QualityIssue). `QualityIssue` struct con Severity (error|warning|info), Tool, Message. `QualityStep(ctx, phase, dag) -> *QualityResult`: (1) verifica que los nodos creados existan en HelixDB, (2) si hay tests, ejecuta `go test ./...` y verifica que pasen, (3) verifica que los archivos modificados existan, (4) compila con `go build ./...`. Retorna QualityResult con issues detallados.
- **Criterio:** Si todos los checks pasan, `Passed=true`. Si un nodo no se creó en HelixDB, issue de error. Tests con mocks.

#### T-5.8: Save Step (save.go)
- **Archivos:** `internal/boomerang/save.go` (nuevo, ~60 líneas)
- **Depende de:** T-5.2, T-4.3, T-4.7
- **Descripción:** `SaveService` struct con memoryStore, embeddingSvc. `SaveResult` struct con FactsSaved, EdgesCreated, Contradictions. `SaveStep(ctx, phase, delegateResult, logData) -> *SaveResult`: (1) extrae decisiones del delegateResult, (2) guarda como Facts con type=decision, (3) crea edges causales entre facts de esta fase y facts anteriores (PRECEDES, DERIVES_FROM), (4) ejecuta DetectContradictions, (5) ejecuta ResolveContradiction (NewestWins), (6) ejecuta ReinforceSalience en facts accedidos.
- **Criterio:** `SaveStep` con delegateResult de 3 tasks guarda ≥3 facts. Detecta y resuelve contradicciones. Tests con store mockeado.

#### T-5.9: Approval refactor (approval.go)
- **Archivos:** `internal/scheduler/approval.go` (refactor, ~40 líneas)
- **Depende de:** T-5.2
- **Descripción:** Refactor completo: reemplazar `PromptApproval()` (stdin) por `ApprovalGate(ctx, phase, summary)` que lanza subagente OpenCode: `opencode subagent zyro-approval-gate --param phase=X --param summary=Y`. Parsear output JSON con campo `approved: bool`. Mantener `PromptApproval` como fallback si OpenCode no está disponible (con log de warning). Eliminar `stdinReader` global.
- **Criterio:** `ApprovalGate(ctx, "F0", "test")` ejecuta `opencode subagent`. Si OpenCode no está, usa fallback stdin. Tests con exec.Command mock.

#### T-5.10: Plugin management (plugin.go)
- **Archivos:** `internal/opencode/plugin.go` (nuevo, ~70 líneas)
- **Depende de:** T-5.2
- **Descripción:** `PluginConfig` struct con ClaudeBridge, LazyLoader, MultiAgent, CustomPaths, Sources ([]SourceConfig). `SourceConfig` struct con Dir, Namespace. `EnsurePluginsConfig(opencodeDir, config)`: verifica que los plugins estén instalados, si no los instala con npm. `WriteBridgePlugin(pluginsDir, config)`: escribe archivo `zyrocli.ts` en pluginsDir con configuración del bridge (sources apuntando a skills Zyro). `WriteLazyLoaderConfig(opencodeDir)`: agrega `opencode-lazy-loader` a plugins de opencode.jsonc.
- **Criterio:** `WriteBridgePlugin("/tmp/plugins", PluginConfig{Sources: []SourceConfig{{Dir: "/tmp/skills", Namespace: "zyro"}}})` escribe archivo TypeScript válido. `EnsurePluginsConfig` verifica existencia.

#### T-5.11: Simplificar opencode/config.go
- **Archivos:** `internal/opencode/config.go` (modificar, ~30 líneas), `internal/opencode/mcptools_embed.go` (modificar, +5 líneas), `internal/opencode/skills_embed.go` (modificar, +5 líneas)
- **Depende de:** T-5.10
- **Descripción:** `config.go`: simplificar para solo manejar perfiles de modelos + plugins. Eliminar lógica de skills embed. `mcptools_embed.go`: agregar docstring `DEPRECATED: MCP tools now load via opencode-lazy-loader`. `skills_embed.go`: agregar docstring `DEPRECATED: skills now load via claude-bridge as .md files`.
- **Criterio:** `go build ./internal/opencode/` compila. Mensajes de deprecación visibles en logs.

#### T-5.12: Integrar Boomerang en scheduler (scheduler.go)
- **Archivos:** `internal/scheduler/scheduler.go` (modificar, ~30 líneas)
- **Depende de:** T-5.2, T-5.9
- **Descripción:** En `Run()`: reemplazar `phase.Run()` directo por `boomerang.RunPhase()`. En `RunPhase()`: si hay BoomerangOrchestrator configurado, usarlo; si no, mantener comportamiento legacy. Agregar campo `Boomerang *BoomerangOrchestrator` a `Config`. Loggear cada paso del ciclo Boomerang con emojis: `→ [1/6 MEMORY]`, `→ [2/6 THINK]`, etc.
- **Criterio:** `Run()` con Boomerang ejecuta ciclo completo. `Run()` sin Boomerang mantiene comportamiento legacy. Logs muestran progreso de 6 pasos.

#### T-5.13: Skills declarativas para bridge
- **Archivos:** 
  - `~/.config/zyro/skills/zyro-orchestrator/SKILL.md` (nuevo, ~30 líneas)
  - `~/.config/zyro/skills/zyro-sdd-apply/SKILL.md` (nuevo, ~25 líneas)
  - `~/.config/zyro/skills/zyro-sdd-verify/SKILL.md` (nuevo, ~25 líneas)
  - `~/.config/zyro/skills/zyro-approval-gate/SKILL.md` (nuevo, ~30 líneas)
- **Depende de:** T-5.10
- **Descripción:** 4 skills declarativas en formato `.md` con frontmatter YAML: name, description, mcp config. `zyro-orchestrator`: orquestador principal del pipeline. `zyro-sdd-apply`: aplica cambios según diseño técnico. `zyro-sdd-verify`: verifica resultados contra criterios. `zyro-approval-gate`: gate de aprobación entre fases — incluye template para preguntar al humano. Cada skill incluye frontmatter con mcp server y contenido markdown con instrucciones.
- **Criterio:** Cada SKILL.md tiene frontmatter YAML válido. `yaml.safe_load()` parsea correctamente. Contenido markdown tiene instrucciones claras.

#### T-5.14: Comando /zyro-approve para OpenCode
- **Archivos:** `~/.config/opencode/commands/zyro-approve.jsonc` (nuevo, ~15 líneas)
- **Depende de:** T-5.13
- **Descripción:** Crear comando slash para OpenCode: `{"command": {"zyro-approve": {"template": "...", "description": "Aprobar fase actual del pipeline SDD", "subtask": false, "agent": "zyro-approval-gate"}}}`. Template incluye placeholders para phase y summary. Agregar en `internal/scheduler/approval.go` la lógica para invocar este comando vía OpenCode CLI.
- **Criterio:** `opencode subagent zyro-approval-gate` ejecuta el skill. Template incluye lugar para phase. JSON es válido.

#### T-5.15: Tests para Boomerang
- **Archivos:** `internal/boomerang/boomerang_test.go` (nuevo, ~180 líneas)
- **Depende de:** T-5.1 a T-5.14
- **Descripción:** Tests: `TestBoomerangOrchestrator_RunPhase`, `TestBoomerangOrchestrator_RunPhase_QualityLoop` (quality falla → redelegate), `TestBoomerangOrchestrator_RunPhase_MaxIterations` (llega a max sin éxito), `TestMemoryStep`, `TestFormatMemoryPrompt`, `TestThinkStep_PhaseF0`, `TestThinkStep_PhaseF3`, `TestGenerateDAGForPhase`, `TestDelegateStep`, `TestDelegateStep_ParallelGroups`, `TestGitStep_Clean`, `TestGitStep_Dirty`, `TestQualityStep_Passed`, `TestQualityStep_Failed`, `TestSaveStep`, `TestApprovalGate` (con exec.Command mock). Mocks: memory.EngramStore, boundari.Enforcer, exec.Command, GitChecker.
- **Criterio:** `go test ./internal/boomerang/ -v` pasa todos. Cobertura >70%.

---

## Grafo de Dependencias

```
Sprint 0 (T-0.1 → T-0.12) [Fundación CLI]
  │
  ├── Sprint 1 (T-1.1 → T-1.10) [Agent-as-Validator]
  │     │
  │     └── Sprint 3 (T-3.1 → T-3.8) [Boundari por Fase]
  │
  └── Sprint 2 (T-2.1 → T-2.12) [HelixDB SDK Go]
        │
        └── Sprint 4 (T-4.1 → T-4.13) [Memoria Causal]
              │
              └── Sprint 5 (T-5.1 → T-5.15) [OpenCode + Boomerang]
```

### Dependencias Internas por Sprint

**Sprint 0:** T-0.1 ← T-0.2 ← T-0.3 ← T-0.4; T-0.5 ← T-0.6 ← T-0.7 ← T-0.8; T-0.9 ← T-0.10; T-0.11 ← T-0.12 (tests al final)
**Sprint 1:** T-1.1 ← T-1.2 ← T-1.3 ← T-1.4 ← T-1.6; T-1.5 ← T-1.6; T-1.7 ← T-1.8; T-1.9 ← T-1.10
**Sprint 2:** T-2.1 ← T-2.2 ← T-2.3 ← T-2.4 ← T-2.5; T-2.6 ← T-2.7; T-2.8 ← T-2.9; T-2.10 ← T-2.12; T-2.11 ← T-2.12
**Sprint 3:** T-3.1 ← T-3.2 ← T-3.3 ← T-3.4 ← T-3.5; T-3.6 ← T-3.7; T-3.8 (al final)
**Sprint 4:** T-4.1 ← T-4.2 ← T-4.3 ← T-4.4; T-4.5 ← T-4.6; T-4.7 ← T-4.8; T-4.9 ← T-4.11; T-4.10 ← T-4.9; T-4.12 ← T-4.13; T-4.14 ← T-4.3; T-4.15 ← T-0.1
**Sprint 5:** T-5.1 ← T-5.2 ← T-5.3 ← T-5.4 ← T-5.5 ← T-5.6 ← T-5.7 ← T-5.8 (secuencial); T-5.9 ← T-5.12; T-5.10 ← T-5.11 ← T-5.13 ← T-5.14; T-5.15 (al final)

---

## Resumen

| Sprint | Tareas | Archivos nuevos | Archivos modif | Líneas est. | Dependencias |
|--------|--------|-----------------|----------------|-------------|--------------|
| S0 | 12 | 7 | 2 | ~800 | Ninguna |
| S1 | 10 | 6 | 6 | ~600 | Sprint 0 |
| S2 | 12 | 7 | 4 | ~700 | Sprint 0 |
| S3 | 8 | 7 | 2 | ~400 | Sprint 1 |
| S4 | 15 | 12 | 2 | ~1100 | Sprint 2 |
| S5 | 15 | 12 | 6 | ~1000 | Sprint 3 + Sprint 4 |
| **Total** | **72** | **51** | **22** | **~4600** | — |

> **Nota:** Algunas tareas marcan +1 archivo en `~/.config/zyro/` o `~/.config/opencode/` (fuera del repo).  
> Esos archivos se escriben en el home del usuario, no en el repositorio.

