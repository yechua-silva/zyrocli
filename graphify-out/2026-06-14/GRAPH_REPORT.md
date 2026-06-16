# Graph Report - .  (2026-06-14)

## Corpus Check
- 0 files · ~0 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1770 nodes · 2407 edges · 95 communities (90 shown, 5 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 214 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 79|Community 79]]
- [[_COMMUNITY_Community 80|Community 80]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 82|Community 82]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 85|Community 85]]
- [[_COMMUNITY_Community 86|Community 86]]
- [[_COMMUNITY_Community 87|Community 87]]
- [[_COMMUNITY_Community 88|Community 88]]
- [[_COMMUNITY_Community 89|Community 89]]
- [[_COMMUNITY_Community 90|Community 90]]
- [[_COMMUNITY_Community 91|Community 91]]
- [[_COMMUNITY_Community 92|Community 92]]

## God Nodes (most connected - your core abstractions)
1. `T` - 37 edges
2. `T` - 31 edges
3. `T` - 25 edges
4. `T` - 25 edges
5. `T` - 21 edges
6. `writeFile()` - 19 edges
7. `Bridge` - 18 edges
8. `T` - 16 edges
9. `tempDir()` - 15 edges
10. `CIO` - 15 edges

## Surprising Connections (you probably didn't know these)
- `writeTestHandoff()` --calls--> `writeFile()`  [INFERRED]
  cmd/zyrocli/init_test.go → internal/skilladvisor/skilladvisor_test.go
- `writeHandoffInTemp()` --calls--> `writeFile()`  [INFERRED]
  cmd/zyrocli/run_test.go → internal/skilladvisor/skilladvisor_test.go
- `writeConventions()` --calls--> `writeFile()`  [INFERRED]
  internal/doc/doc_test.go → internal/skilladvisor/skilladvisor_test.go
- `TestUpdateGraph_NoSignificantChange()` --calls--> `writeFile()`  [INFERRED]
  internal/doc/doc_test.go → internal/skilladvisor/skilladvisor_test.go
- `TestUpdateGraph_SignificantChange()` --calls--> `writeFile()`  [INFERRED]
  internal/doc/doc_test.go → internal/skilladvisor/skilladvisor_test.go

## Import Cycles
- None detected.

## Communities (95 total, 5 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (64): Phase, Config, Context, Phase, Result, Config, Context, Phase (+56 more)

### Community 1 - "Community 1"
Cohesion: 0.05
Nodes (69): DiscoveryQuery, Payload, SkillEntry, ValidatedSkill, DiscoveryQuery, Payload, DiscoveryQuery, ScoreResult (+61 more)

### Community 2 - "Community 2"
Cohesion: 0.08
Nodes (51): Conventions, tempDir(), TestExport_GeneratesFiles(), TestExport_NilIndex(), TestGenerateIndex_EmptyConventions(), TestGenerateIndex_MissingConventions(), TestGenerateIndex_WithChanges(), TestGenerateIndex_WithProjectKeys() (+43 more)

### Community 3 - "Community 3"
Cohesion: 0.08
Nodes (41): Feature, T, Decomposer, estimateComplexity(), FeatureByID(), inferDependencies(), NewDecomposer(), splitStory() (+33 more)

### Community 4 - "Community 4"
Cohesion: 0.10
Nodes (40): ContractResult, Context, T, Contract, ContractExecutor, ContractResult, NewContractExecutor(), TestContractExecutor_ExecuteBatch() (+32 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (37): Advisory, Report, Report, T, T, Advisor, NewAdvisor(), AdvisorConfig (+29 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (37): FS, FuncMap, Config, ValidatedSkill, T, T, Config, Renderer (+29 more)

### Community 7 - "Community 7"
Cohesion: 0.08
Nodes (32): Cmd, Bridge, NewBridge(), TestBridge_IsRunning_NotStarted(), TestBridge_QueryDocs_NotRunning(), TestBridge_QueryDocs_PipeMock(), TestBridge_ResolveLibraryID_EmptyID(), TestBridge_ResolveLibraryID_NotRunning() (+24 more)

### Community 8 - "Community 8"
Cohesion: 0.05
Nodes (40): 10. Migration / Rollout, 11. Open Questions, 1. Component Diagram, 2. Macro 1 — Documentación Engram puro, 2a. Topic Keys para CADA artifact, 2b. Formato estándar de entry en Engram, 2c. Protocolo de búsqueda estructurada, 2d. zyro-doc-search tool (+32 more)

### Community 9 - "Community 9"
Cohesion: 0.12
Nodes (33): PoolConfig, Result, ResultStatus, Runner, errorsIs(), NewRunner(), atomicMax(), safeTask() (+25 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (36): Requirement: Config-Driven Governance, Requirement: F2–F4 — Real Phase Implementations, Requirement: PhaseRunner Interface, Delta for Scheduler Engine, MODIFIED Requirements, Requirement: F1-F4 Real Implementations via MacroPhaseRunner, Requirement: F1 Phase — Real Skill Advisor Integration, Requirement: GuidedApproval — Contextual Dialog (+28 more)

### Community 11 - "Community 11"
Cohesion: 0.05
Nodes (36): Purpose, Requirement: BuildDiscoveryQuery, Requirement: detectFramework, Requirement: Deterministic Weighted Scoring, Requirement: DiscoverAndRank (Unified Entry Point), Requirement: extractKeywords, Requirement: Merge Local and API Results, Requirement: Pipeline Type Definitions (+28 more)

### Community 12 - "Community 12"
Cohesion: 0.06
Nodes (31): 1. No hay definición de "investigación", 2. No hay flujo de error, 3. No hay métricas de éxito, 4. No hay rollbacks, 5. Doc tools sin especificación, 6. Chained PRs sin métrica, Acciones inmediatas (antes de arrancar SDD):, Archivos referenciados (+23 more)

### Community 13 - "Community 13"
Cohesion: 0.07
Nodes (29): CLI, doc-tools (internal/doc/), final integration:, handoff adjustments:, Integration, Phase 1.1: Skill Advisor Core, Phase 1.2: Context MCP Bridge, Phase 1.3: Tests (+21 more)

### Community 14 - "Community 14"
Cohesion: 0.15
Nodes (25): Parse(), TestIntegrationParseValidateFull(), TestIntegrationParseValidateMinimal(), TestParseFile(), TestParseFileNotFound(), TestParseInvalidYAML(), TestParseStdin(), TestParseStdinEmpty() (+17 more)

### Community 15 - "Community 15"
Cohesion: 0.07
Nodes (29): Handoff Parser Specification, Purpose, Requirement: Business Validation, Requirement: Error Handling, Requirement: Handoff Structs, Requirement: Stdin Pipe, Requirement: YAML Parsing, Requirements (+21 more)

### Community 16 - "Community 16"
Cohesion: 0.07
Nodes (28): Architecture Overview, cmd/zyrocli/run.go (modificado), Component Details, Data Flow, Design: Skill Validation Pipeline, discover.go (modificado — deprecado, sin cambios HTTP), Discover.py Failure (graceful degradation), Error Handling (+20 more)

### Community 17 - "Community 17"
Cohesion: 0.07
Nodes (28): Requirement: AGENT.md Reflects Macro Fases 1-4, Requirement: Build Verification Extended, Requirement: .zyro/conventions.yaml Generation, Requirement: .zyro/ Directory Created, Scenario: AGENT.md describes macro fases, Scenario: AGENT.md lists all 8 SDD skills, Scenario: All 10 files created, Scenario: Conventions file has topic keys (+20 more)

### Community 18 - "Community 18"
Cohesion: 0.07
Nodes (27): Contract Testing Specification, Purpose, Requirements, Requirement: Contract Types, Requirement: ContractExecutor with Given/When/Then, Requirement: Diff Threshold, Requirement: ExecuteBatch, Requirement: GraphifyDiff for Structural Comparison (+19 more)

### Community 19 - "Community 19"
Cohesion: 0.07
Nodes (26): Delta Spec: Contract Testing, NEW Requirements, Requirement: Contract Types, Requirement: ContractExecutor with Given/When/Then, Requirement: Diff Threshold, Requirement: ExecuteBatch, Requirement: GraphifyDiff for Structural Comparison, Requirement: Report Aggregation (+18 more)

### Community 20 - "Community 20"
Cohesion: 0.07
Nodes (26): ADDED Requirements, MODIFIED Requirements, REMOVED Requirements, Requirement: Cobra Subcommand, Requirement: Exit Codes, Requirement: Handoff Parse Into Scaffold Config, Requirement: Missing handoff.yaml Error, Requirement: OpenCode PATH Check Before Launch (+18 more)

### Community 21 - "Community 21"
Cohesion: 0.08
Nodes (24): Architecture Decisions, Current Code (antes), Data Flow, Decision: Lenient opencode check (warn, don't fail), Decision: Remove --phase flag entirely, Decision: Reuse init.go's scaffold pattern verbatim, Dependencies, Design: fix-zyrocli-run-scaffold (+16 more)

### Community 22 - "Community 22"
Cohesion: 0.09
Nodes (22): Apply Runner Specification, Purpose, Requirements, Requirement: Context Propagation, Requirement: Fail-Fast Error Handling, Requirement: Per-Task Timeout, Requirement: Result Collection via Channel, Requirement: Task Runner with Goroutine Pool (+14 more)

### Community 23 - "Community 23"
Cohesion: 0.09
Nodes (22): Investigation Macro Specification, Purpose, Requirement: Advisor, Requirement: Context Documentation Lookup, Requirement: GitMCP Repository Analysis, Requirement: Investigation Report, Requirement: ResearchEngine, Requirement: Web Fetch for External Context (+14 more)

### Community 24 - "Community 24"
Cohesion: 0.09
Nodes (21): Delta Spec: Apply Runner, NEW Requirements, Requirement: Context Propagation, Requirement: Fail-Fast Error Handling, Requirement: Per-Task Timeout, Requirement: Result Collection via Channel, Requirement: Task Runner with Goroutine Pool, Requirement: Task Types (+13 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (21): Doc Tools Specification, Purpose, Requirement: CLI Sync Command, Requirement: Conventions YAML, Requirement: Doc Index Generation, Requirement: Export Templates, Requirement: Graph Diff, Requirement: Index Search Protocol (+13 more)

### Community 26 - "Community 26"
Cohesion: 0.10
Nodes (20): Change context, Delta scenarios, Delta Spec: doc-tools, New requirements, Parent spec, R-DOC-004: Doc Index Generation, R-DOC-005: Index Search Protocol, R-DOC-006: Sync Cycle (+12 more)

### Community 27 - "Community 27"
Cohesion: 0.10
Nodes (19): Build & Vet, Help output (`go run ./cmd/zyrocli init --help`), init command (`cmd/zyrocli/...`), Issues Found, Overall Verdict: ✅ PASS, R1: Scaffold Flag — ✅ PASS, R2: Project Directory Generation — ✅ PASS, R3: AGENT.md Generation — ✅ PASS (with warning) (+11 more)

### Community 28 - "Community 28"
Cohesion: 0.20
Nodes (13): Client, DiscoverResult, Duration, SkillEntry, Time, RWMutex, cacheEntry, Discover() (+5 more)

### Community 29 - "Community 29"
Cohesion: 0.11
Nodes (17): Architecture Decisions, Data Flow, Decision: Raw bytes, no template rendering, Decision: Scripts as files, not empty dir markers, Decision: Separate embed.FS for scripts, Design: python-tools-scaffold, explorer.py args, File Changes (+9 more)

### Community 30 - "Community 30"
Cohesion: 0.11
Nodes (17): CIO DSL Compiler Specification, Purpose, Requirement: CIO Struct Completeness, Requirements, Scenario: All sections populated, Scenario: Zero-value safety, Requirement: CIO Compile — Engram Key Emission, Requirement: `EngramEntry` Output Struct (+9 more)

### Community 31 - "Community 31"
Cohesion: 0.24
Nodes (10): Context, Time, DocQueryFn, DocSource, GitSource, Report, ResearchEngine, ResearchEngineConfig (+2 more)

### Community 32 - "Community 32"
Cohesion: 0.11
Nodes (16): Purpose, Removed Requirements, Requirement: Cobra Subcommand, Requirement: Doc.Sync() Post-Scaffold Hook, Requirement: Exit Codes, Requirement: Handoff Parse Into Scaffold Config, Requirement: Missing handoff.yaml Error, Requirement: OpenCode PATH Check Before Launch (+8 more)

### Community 33 - "Community 33"
Cohesion: 0.12
Nodes (16): Context MCP Bridge Specification, LibraryID Type, Purpose, Requirement: JSON-RPC Query, Requirement: Library ID Resolution, Requirement: Process Lifecycle, Requirements, Scenario: Force kill on timeout (+8 more)

### Community 34 - "Community 34"
Cohesion: 0.21
Nodes (8): Config, Context, Phase, Result, F1Runner, F2Runner, F3Runner, F4Runner

### Community 35 - "Community 35"
Cohesion: 0.12
Nodes (15): Affected Areas, Approach, Capabilities, Dependencies, Effort Estimate, In Scope, Intent, Modified Capabilities (+7 more)

### Community 36 - "Community 36"
Cohesion: 0.12
Nodes (15): Change Context, Delta Spec: Investigation (Macro 1), Modified Requirements, New Requirements, R-INV-003: Investigation Report — Output Path, R-INV-004: ResearchEngine, R-INV-005: Investigation Report, R-INV-006: Advisor (+7 more)

### Community 37 - "Community 37"
Cohesion: 0.12
Nodes (15): Scenario: Bullet list decomposition, Scenario: Circular dependency, Scenario: Empty story rejected, Scenario: Linear dependency chain, Scenario: Parallel independent features, Scenario: Unknown dependency, Change Context, Delta Spec: Planning (Macro 2) (+7 more)

### Community 38 - "Community 38"
Cohesion: 0.12
Nodes (15): Requirement: AGENT.md Reflects Macro Fases 1-4, Requirement: Build Verification Extended, Requirement: .zyro/conventions.yaml Generation, Requirement: .zyro/ Directory Created, Scenario: AGENT.md describes macro fases, Scenario: AGENT.md lists all 8 SDD skills, Scenario: All 10 files created, Scenario: Conventions file has topic keys (+7 more)

### Community 39 - "Community 39"
Cohesion: 0.12
Nodes (15): Scenario: Bullet list decomposition, Scenario: Circular dependency, Scenario: Empty story rejected, Scenario: Linear dependency chain, Scenario: Parallel independent features, Scenario: Unknown dependency, Planning Macro Specification, Purpose (+7 more)

### Community 40 - "Community 40"
Cohesion: 0.13
Nodes (14): Architecture Decisions, Data Flow, Decision: Multi-Error Validation with `errors.Join()`, Decision: Parse/Validate Separation, Decision: Stdin via `"-"` Convention, Decision: Struct Design — No Pointers, `omitempty`, Decision: Subcommand in Separate File, Design: Implementar parser de handoff.yaml + comando init (+6 more)

### Community 41 - "Community 41"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 42 - "Community 42"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 43 - "Community 43"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 44 - "Community 44"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 45 - "Community 45"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 46 - "Community 46"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 47 - "Community 47"
Cohesion: 0.27
Nodes (13): CIO, T, Compile(), GenerateTopicKey(), TestCompile_EmptyCIO(), TestCompile_FullCIO(), TestCompile_NilCIO(), TestCompile_ZeroValueSafety() (+5 more)

### Community 48 - "Community 48"
Cohesion: 0.13
Nodes (14): ADDED Requirements, Requirement: Capabilities Field in Payload, Requirement: Dependencies Field in Payload, Requirement: Handoff Structs Extended, Scenario: Capabilities and dependencies round-trip, Scenario: Capabilities optional (absent), Scenario: Dependencies optional (absent), Delta for Handoff Parser (+6 more)

### Community 49 - "Community 49"
Cohesion: 0.13
Nodes (14): Affected Areas, Approach, Capabilities, Dependencies, In Scope, Intent, Modified Capabilities, New Capabilities (+6 more)

### Community 50 - "Community 50"
Cohesion: 0.14
Nodes (13): Architecture Decisions, Data Flow, Decision: Cobra root command with persistent flags, Decision: Compilable stubs vs empty files, Decision: Dependencies — cobra + yaml.v3 only, Decision: Flat internal/ packages (no nesting), Design: Inicializar módulo Go y estructura base del proyecto, File Changes (+5 more)

### Community 51 - "Community 51"
Cohesion: 0.14
Nodes (13): 1. skill-advisor spec, 2. scheduler-engine spec, 3. zyrocli-run spec, 4. Tests, 5. Errores en consola, CRITICAL (must fix), Files Verified, Status: PASS (+5 more)

### Community 52 - "Community 52"
Cohesion: 0.14
Nodes (13): Comandos CLI, Contrato handoff.yaml (input de Holdin Admin), Estructura del proyecto, F1: Planificación, F2: Especificación, F3: Implementación, F4: Cierre, Flujo de 4 macro-fases (scheduler state machine) (+5 more)

### Community 53 - "Community 53"
Cohesion: 0.14
Nodes (13): Delta for CIO DSL Compile, MODIFIED Requirements, Requirement: CIO Compile — Engram Key Emission, Requirement: `EngramEntry` Output Struct, Requirement: Markdown Serialization, Requirement: Stable Topic Key Generation, Scenario: All sections serialized, Scenario: Empty contract name uses "unnamed" fallback (+5 more)

### Community 54 - "Community 54"
Cohesion: 0.15
Nodes (13): Constraint, Contract, IOMethod, Operation, Rule, CIO, Constraint, Contract (+5 more)

### Community 55 - "Community 55"
Cohesion: 0.14
Nodes (13): Purpose, Python Tools Specification, Requirement: AGENT.md Enforcement Table, Requirement: AGENT.md Enforcement Table Test, Requirement: Explorer Script, Requirement: Linter Script, Requirement: Python Runtime Integration Test, Requirement: ReadScript Helper (+5 more)

### Community 56 - "Community 56"
Cohesion: 0.15
Nodes (12): Delivery Macro Specification, Purpose, Requirement: Chained PR Splitting, Requirement: PR Creation, Requirement: Work Unit Commits, Requirements, Scenario: Commit structure, Scenario: gh not installed (+4 more)

### Community 57 - "Community 57"
Cohesion: 0.15
Nodes (12): ADDED Requirements, MODIFIED Requirements, Requirement: Doc.Sync() Post-Scaffold Hook, Requirement: Help Text Documents Doc.Sync(), Requirement: Scaffold Messages Updated, Delta for Zyrocli Run CLI, Scenario: Doc count shown in output, Scenario: Doc sync output shown (+4 more)

### Community 58 - "Community 58"
Cohesion: 0.55
Nodes (11): T, ensureNoOpencode(), runCmdDirect(), TestRunDirectoryCollision(), TestRunDocSyncIntegration(), TestRunHelp(), TestRunMissingHandoff(), TestRunOpenCodeMissing() (+3 more)

### Community 59 - "Community 59"
Cohesion: 0.18
Nodes (10): Phase 1: Foundation — Structs v2.0, Phase 2: Parser, Phase 3: Validación, Phase 4: CLI, Phase 5: Configs, Phase 6: Tests, Phase 7: Verification, Review Workload Forecast (+2 more)

### Community 60 - "Community 60"
Cohesion: 0.45
Nodes (10): Buffer, T, runInitCmd(), TestInitNoFlags(), TestInitOpenCodeWithoutScaffold(), TestInitScaffoldFlag(), TestInitScaffoldFlagsParsed(), TestInitScaffoldOpenCode() (+2 more)

### Community 61 - "Community 61"
Cohesion: 0.18
Nodes (10): Purpose, Python Tools Specification, Requirement: AGENT.md Enforcement Table, Requirement: Explorer Script, Requirement: Linter Script, Requirement: Scaffold Integration, Requirement: Script Bundle, Requirement: Test Coverage (+2 more)

### Community 62 - "Community 62"
Cohesion: 0.18
Nodes (10): Design: scaffold-opencode-init, File Changes, Interfaces / Contracts, Key Types, Migration / Rollout, Package Architecture, Resolved Decisions, Sequence Diagram (+2 more)

### Community 63 - "Community 63"
Cohesion: 0.18
Nodes (10): Purpose, Requirement: AGENT.md Generation, Requirement: Error Handling, Requirement: opencode.json Generation, Requirement: OpenCode Launch, Requirement: Project Directory Generation, Requirement: Scaffold Flag, Requirement: Testing (+2 more)

### Community 64 - "Community 64"
Cohesion: 0.18
Nodes (10): Purpose, Requirement: Config-Driven Governance, Requirement: F2–F4 — Real Phase Implementations, Requirement: PhaseRunner Interface, Requirements, Scheduler Engine Specification, Requirement: F1 Phase — Unified Skill Discovery with DiscoverAndRank, Requirement: GuidedApproval Gates (+2 more)

### Community 65 - "Community 65"
Cohesion: 0.20
Nodes (9): Architecture Decisions, Data Flow, Design: scheduler-zyrocli-run, File Changes, Interfaces / Contracts, Migration / Rollout, Open Questions, Technical Approach (+1 more)

### Community 66 - "Community 66"
Cohesion: 0.20
Nodes (9): Phase 1: Foundation — Type Definitions, Phase 2: Core Logic — Query, Verify, Score, Phase 3: Python Discovery + Go Bridge, Phase 4: Embed + Scaffold + Callers, Phase 5: Testing + Verification, Review Workload Forecast, Suggested Work Units, Tasks: Skill Validation Pipeline (+1 more)

### Community 67 - "Community 67"
Cohesion: 0.24
Nodes (10): ApprovalPoint, Governance, Limits, MVP, Payload, Project, Source, Testing (+2 more)

### Community 68 - "Community 68"
Cohesion: 0.29
Nodes (8): SkillEntry, DiscoveryQuery, Layer, ScoreResult, Skill, SkillQuery, ValidatedSkill, ValidationError

### Community 69 - "Community 69"
Cohesion: 0.20
Nodes (9): Requirement: AGENT.md Enforcement Table Test, Requirement: Python Runtime Integration Test, Requirement: ReadScript Helper, Modified Requirements, New Requirements, Python Tools Specification — Delta, Requirement: Linter Script (output shape + no-config behavior), Requirement: Script Bundle (exit behavior) (+1 more)

### Community 70 - "Community 70"
Cohesion: 0.20
Nodes (9): Purpose, Requirement: Config-Driven Governance, Requirement: PhaseRunner Interface, Requirements, Scheduler Engine Specification, Requirement: Approval Gates, Requirement: F1 Phase Stub, Requirement: F2–F4 Stubs (+1 more)

### Community 71 - "Community 71"
Cohesion: 0.20
Nodes (9): Purpose, Requirement: Cobra Subcommand, Requirement: Exit Codes, Requirement: Missing handoff.yaml Error, Requirements, Zyrocli Run CLI Specification, Requirement: Auto Mode Flag, Requirement: Flag Mutual Exclusion (+1 more)

### Community 72 - "Community 72"
Cohesion: 0.22
Nodes (8): Design: fix-python-tools-compliance, File Changes, linter.py changes, scaffold_test.go changes, scripts.go changes, Technical Approach, test-runner.py changes, Testing Strategy

### Community 73 - "Community 73"
Cohesion: 0.22
Nodes (8): Estimated changed lines: 80-120, Review Workload Forecast, Task 1: Fix linter.py, Task 2: Fix test-runner.py, Task 3: Add ReadScript() helper, Task 4: Add Python runtime integration test, Task 5: Add enforcement table assertion, Tasks: fix-python-tools-compliance

### Community 74 - "Community 74"
Cohesion: 0.22
Nodes (8): Phase 1: Foundation, Phase 2: Entry Point, Phase 3: Internal Stubs, Phase 4: Configs, Phase 5: Verification, Review Workload Forecast, Suggested Work Units, Tasks: Inicializar módulo Go y estructura base del proyecto

### Community 75 - "Community 75"
Cohesion: 0.22
Nodes (8): Build & Tests Execution, Coherence (Design), Completeness, Correctness (Static Evidence), Issues Found, Spec Compliance Matrix, Verdict, Verification Report

### Community 76 - "Community 76"
Cohesion: 0.22
Nodes (8): Phase A: Scaffold Package, Phase B: Templates, Phase C: CLI Integration, Phase D: Tests, Phase E: Verify, Review Workload Forecast, Suggested Work Units, Tasks: scaffold-opencode-init

### Community 77 - "Community 77"
Cohesion: 0.25
Nodes (7): Phase 1: Schema & Types (Foundation), Phase 2: Scheduler Engine (Core), Phase 3: CLI Integration (Wiring), Phase 4: Testing (Verification), Review Workload Forecast, Suggested Work Units, Tasks: scheduler-zyrocli-run

### Community 78 - "Community 78"
Cohesion: 0.29
Nodes (6): Approach, In Scope, Intent, Out of Scope, Proposal: fix-python-tools-compliance, Scope

### Community 79 - "Community 79"
Cohesion: 0.29
Nodes (6): Phase 1: Limpieza de imports y variables obsoletas, Phase 2: Eliminar lógica del pipeline SDD, Phase 3: Insertar scaffold+opencode flow, Phase 4: Eliminar flag --phase, Phase 5: Verificación, Tasks: fix-zyrocli-run-scaffold

### Community 80 - "Community 80"
Cohesion: 0.29
Nodes (6): Archive Contents, Archive Report: skill-validation-pipeline, SDD Cycle Complete, Source of Truth Updated, Specs Synced, Verification Summary

### Community 81 - "Community 81"
Cohesion: 0.38
Nodes (7): Duration, Config, ValidatedSkill, Phase, PhaseRunner, Result, Status

### Community 82 - "Community 82"
Cohesion: 0.43
Nodes (6): main(), parse_coverage(), parse_errors(), Parse stderr into structured error list [{file, test, message}]., Parse coverage.out into structured format: lines and percent as numbers., run()

### Community 83 - "Community 83"
Cohesion: 0.33
Nodes (5): Contract, Loading protocol, Skill Registry — ZyroAgentCLI, Skills, Sources scanned

### Community 84 - "Community 84"
Cohesion: 0.33
Nodes (5): Configuration, Context MCP Bridge — Delta Spec, Implementation references, Key additions, What changed

### Community 85 - "Community 85"
Cohesion: 0.33
Nodes (5): Implementation references, Key additions, Scoring weights, Skill Advisor — Delta Spec, What changed

### Community 89 - "Community 89"
Cohesion: 0.50
Nodes (3): Delta for Handoff Parser, MODIFIED Requirements, Requirement: CLI Integration

### Community 90 - "Community 90"
Cohesion: 0.67
Nodes (3): discover(), main(), Query the skills.sh API and return a list of skill dicts.

## Knowledge Gaps
- **815 isolated node(s):** `Buffer`, `Payload`, `WaitGroup`, `CancelFunc`, `Task` (+810 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `writeFile()` connect `Community 1` to `Community 2`, `Community 6`, `Community 14`, `Community 58`, `Community 60`?**
  _High betweenness centrality (0.027) - this node is a cross-community bridge._
- **Why does `Parse()` connect `Community 14` to `Community 0`, `Community 34`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **What connects `Buffer`, `Payload`, `WaitGroup` to the rest of the system?**
  _818 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05524537173082574 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.05297334244702666 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.08469449485783424 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.08470588235294117 - nodes in this community are weakly interconnected._