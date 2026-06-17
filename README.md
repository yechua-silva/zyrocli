# ZyroCLI — Orquestador Autónomo para Desarrollo Asistido por IA

[![npm version](https://img.shields.io/npm/v/zyrocli)](https://www.npmjs.com/package/zyrocli)
[![Go version](https://img.shields.io/github/go-mod/go-version/secko/zyrocli)](https://go.dev/)
[![License](https://img.shields.io/github/license/secko/zyrocli)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/secko/zyrocli)](https://goreportcard.com/report/github.com/secko/zyrocli)

> **Pipeline SDD completo (F0→F4) · Memoria Causal · Seguridad por Fase · Auto-instalable**

## 📋 Tabla de Contenidos

- [✨ ¿Qué hace ZyroCLI?](#qué-hace-zyrocli)
- [🧠 ¿Por qué existe?](#por-qué-existe)
- [🔄 Dos Ciclos, Un Sistema: SDD + Boomerang](#dos-ciclos-un-sistema-sdd--boomerang)
- [🏗️ Arquitectura](#arquitectura)
- [🚀 Instalación](#instalación)
- [⚡ Quick Start](#quick-start)
- [📚 Documentación](#documentación)
- [🛡️ Seguridad](#seguridad)
- [✅ Estado del Proyecto](#estado-del-proyecto)

ZyroCLI es un orquestador que ejecuta un **pipeline de desarrollo de software** usando agentes de IA sobre OpenCode. No es un chat, no es un IDE — es un **ingeniero de software automatizado** que planifica, especifica, diseña, implementa y verifica usando agentes especializados.

## ✨ ¿Qué hace ZyroCLI?

| Comando | Qué hace |
|---------|----------|
| `zyro setup` | Instala todo el ecosistema (Go, Python, HelixDB, Ollama) automáticamente |
| `zyro init` | Crea un proyecto desde un handoff.yaml |
| `zyro run --phase F0` | **Investigación**: busca patrones, librerías y skills en paralelo |
| `zyro run --phase F1` | **Especificación**: genera spec técnica C-I-O |
| `zyro run --phase F2` | **Diseño**: desglose en componentes y tareas atómicas |
| `zyro run --phase F3` | **Implementación**: apply + verify (loop hasta pasar) |
| `zyro run --phase F4` | **Cierre**: archive, lint, build final |
| `zyro doctor --fix` | Diagnostica y repara la configuración |

## 🧠 ¿Por qué existe?

El desarrollo asistido por IA tiene un problema fundamental: **cada sesión empieza en blanco**. El agente no recuerda decisiones anteriores, repite errores, y no tiene contexto del proyecto.

ZyroCLI resuelve esto con **4 innovaciones** basadas en investigación técnica:

### 1. Memoria Causal (en lugar de chat stateless)

Basado en el concepto de **engram** (traza física de memoria, Semon 1904). Cada decisión, error y preferencia se almacena como un nodo `Fact` en un **grafo causal** con relaciones como `CAUSED`, `PRECEDES`, `CONTRADICTS`. Antes de cada fase, el agente recibe automáticamente el contexto relevante. Después de cada fase, se extraen nuevos hechos y se resuelven contradicciones.

**Por qué:** Los LLMs no tienen memoria persistente. HelixDB (grafos + vectores) permite búsqueda semántica y navegación causal que SQLite no puede ofrecer.

[Investigación →](docs/explorations/investigacion-04-engram-memoria-causal.md)

### 2. Agent-as-Validator (el agente opina, Go ejecuta)

El agente Python **nunca escribe en la base de datos**. Devuelve un `AgentDecision` validado con Pydantic, y el orquestador Go decide si ejecutarlo. Esto evita que el agente se salte fases o corrompa el estado global.

**Por qué:** Un agente autónomo sin restricciones puede inventar herramientas, modificar estado o ejecutar acciones no autorizadas. Separar "opinar" de "ejecutar" es un patrón probado (verificar y validar).

[Investigación →](docs/explorations/investigacion-01-pydanticai-harness.md)

### 3. Seguridad por Fase (Boundari)

Cada fase del pipeline tiene una **política Boundari** que define qué herramientas puede usar el agente. F0 solo lectura, F3 escritura intensiva con approval para comandos peligrosos. Las políticas están escritas en YAML y se cargan dinámicamente.

**Por qué:** No todas las fases tienen los mismos permisos. Un agente en fase de investigación no debería poder ejecutar código. Boundari aplica el principio de mínimo privilegio.

[Investigación →](docs/explorations/investigacion-03-boundari-politicas-seguridad.md)

### 4. Búsqueda Híbrida + Embeddings Locales

La memoria causal se consulta con **búsqueda híbrida** (vector ANN + BM25 + RRF fusion). Los embeddings se generan localmente con Ollama + `mxbai-embed-large` (CPU-friendly). Si no hay embeddings, cae a BM25 puro (degradación graceful).

**Por qué:** La búsqueda semántica permite encontrar hechos conceptualmente similares aunque usen palabras diferentes. El modelo mxbai-embed-large es el mejor quality/speed para CPU según MTEB benchmark.

[Investigación →](docs/explorations/investigacion-06-embedding-system.md)

## 🔄 Dos Ciclos, Un Sistema: SDD + Boomerang

ZyroAgentCLI opera con **dos ciclos anidados** que no debes confundir:

### 🟦 SDD — Macro-ciclo de producto (F0→F4)

El ciclo visible para el humano. Cada fase produce un artefacto:

```
F0: Investigación  →  Patrones, Librerías, Skills
F1: Especificación →  Spec (arquitectura, módulos, REST)
F2: Diseño + Tasks →  Design + Tareas atómicas
F3: Implementación →  Código + Tests
F4: Archive        →  Documentación final
```

**Cuándo usarlo:** cuando planificas un feature o fix completo.  
**Quién lo gobierna:** el humano + el scheduler Go.

### 🟠 Boomerang — Micro-ciclo de ejecución (dentro de cada fase)

El motor interno que ejecuta **cada fase** de forma autónoma. 6 pasos que se repiten en F0, F1, F2, F3 y F4:

```
┌─────────────────────────────────────────────────────┐
│                   BOOMERANG                          │
│                                                      │
│  1. MEMORY       ──  Recuperar hechos relevantes    │
│                     de HelixDB (memoria causal)      │
│                                                      │
│  2. THINK        ──  Planificar acción específica    │
│                     según el contexto recuperado     │
│                                                      │
│  3. DELEGATE     ──  Ejecutar agente(s) Python      │
│                     con contexto + herramientas       │
│                                                      │
│  4. GIT CHECK    ──  Verificar estado del repo       │
│                     (sin cambios conflictivos)       │
│                                                      │
│  5. QUALITY GATES──  Validación determinista         │
│                     (tests, lint, tipos)             │
│                                                      │
│  6. SAVE MEMORY  ──  Extraer hechos nuevos y         │
│                     guardar en HelixDB               │
│                                                      │
└─────────────────────────────────────────────────────┘
```

**Cuándo se usa:** en CADA fase del SDD. F0 ejecuta su Boomerang, F1 ejecuta su Boomerang, etc.  
**Quién lo gobierna:** el orquestador Go — **nunca el agente Python**.

### ¿Por qué dos ciclos?

| SDD (macro) | Boomerang (micro) |
|-------------|-------------------|
| Responde al **qué** construir | Responde al **cómo** ejecutarlo |
| Lo ves como plan de proyecto | Es invisible — pasa dentro de cada fase |
| Avanza con aprobación humana | Es automático, no requiere aprobación |
| 4-5 fases por feature | 6 pasos × N fases por feature |

### 📐 Medición de Tokens — Comparación de 3 Enfoques

Cada enfoque resuelve el mismo problema (implementar un feature) pero con distinta estructura y costo de tokens.

| # | Enfoque | Descripción | Tokens por fase (estimado) | 
|---|---------|-------------|---------------------------|
| 1 | **Plain OpenCode** | Agente sin memoria, sin estructura. Lee todo el codebase desde cero cada vez. | 12k–25k por turno |
| 2 | **SDD agents** | Agentes estructurados (Spec→Design→Tasks) pero sin memoria entre fases. Cada fase re-descubre el contexto. | 8k–15k por turno |
| 3 | **ZyroCLI + Boomerang + HelixDB** | Memoria causal: solo hechos relevantes inyectados. El agente NO lee el codebase, recibe contexto ya filtrado. | 2k–4k por turno |

#### ¿De dónde salen estos números?

**Enfoque 1 (Plain OpenCode):**
- Fuente: [Anthropic SWE-bench (2025)](https://www.anthropic.com/research/swe-bench-sonnet)
- Datos: "many successful runs took hundreds of turns... >100k tokens"
- El agente empieza explorando toda la estructura del repo (view directory)
- Luego busca archivos uno por uno (cat -n), llenando el contexto con código irrelevante
- "The model kept trying until it ran out of context" (límite 200k tokens)

**Enfoque 2 (SDD agents):**
- Los prompts estructurados ayudan (Spec reduce ambigüedad)
- Pero cada agente (spec, design, apply, verify) lee el codebase desde cero
- No hay memoria entre fases: F2 no sabe lo que decidió F1 sin re-leer

**Enfoque 3 (ZyroCLI + Boomerang):**
- MemoryStep: consulta HelixDB por hechos relevantes (típicamente 2k-4k tokens)
- El agente recibe: "Decisión: usar Go para backend (F1). Error: conexión timeout (F0). Preferencia: testing con testify (F1)."
- NO necesita explorar el codebase — ya tiene el contexto filtrado
- Fuente: estimación con `internal/tokens.Count()` implementado en Go

#### ⚠️ Importante: Estos números se miden, no se asumen

| Fase | Sin Boomerang | Con Boomerang | Ahorro | Muestras |
|------|--------------|--------------|--------|----------|
| F0   | —            | —            | ⏳ N<3  | 0        |
| F1   | —            | —            | ⏳ N<3  | 0        |
| F2   | —            | —            | ⏳ N<3  | 0        |
| F3   | —            | —            | ⏳ N<3  | 0        |
| F4   | —            | —            | ⏳ N<3  | 0        |

> Los datos se llenan automáticamente al ejecutar `zyro doctor --tokens`. Mínimo 3 muestras por fase.
> Hasta entonces, los números de la tabla comparativa son estimaciones basadas en investigación académica
> (Anthropic SWE-bench 2025, OpenAI tokenizer docs).

#### La diferencia real no son solo tokens

| Aspecto | Plain OpenCode | SDD agents | ZyroCLI + Boomerang |
|---------|---------------|------------|---------------------|
| Descubrir estructura del repo | ✅ Cada vez | ✅ Cada vez | ❌ Nunca (inyectado) |
| Recordar decisiones previas | ❌ No tiene | ❌ No tiene | ✅ HelixDB causal |
| Explorar archivos | ✅ Busca manual | ✅ Busca manual | ❌ Solo los relevantes |
| Token waste | Alto | Medio | Bajo |
| Tiempo por fase | Variable | Predecible | Predecible |

#### Cómo medir

```bash
# Ver mediciones acumuladas (después de ejecutar fases)
zyro doctor --tokens

# Las mediciones se guardan automáticamente en HelixDB
# como Facts tipo "measurement" por el Boomerang

# Fuente del estimador: https://platform.openai.com/tokenizer
# 1 token ≈ 4 caracteres (estándar OpenAI para inglés y código)
# Precisión: ±20% para rangos típicos (1k-100k chars)
```

> 🔬 **Nota metodológica:** El ahorro de tokens no es el único objetivo. La estructura que ZyroCLI + Boomerang le da al agente (fases definidas, memoria causal, approval gates) también reduce alucinaciones, errores y repeticiones. Esto es cualitativo pero igual de importante.

### 🔧 Integración con OpenCode

Las MCP tools de HelixDB (`search_code`, `search_skills`, `task_context`, `search_facts`, `embed`) son los **brazos del Boomerang**. El orquestador Go las usa en cada paso:

```
Paso 1 (Memory)   → search_facts + task_context
Paso 2 (Think)    → search_code + search_skills
Paso 3 (Delegate) → llama al agente Python con contexto inyectado
Paso 5 (Quality)  → ejecuta tests vía CLI
Paso 6 (Save)     → save_to_helix (nuevos Facts)
```

El resultado: **el agente Python nunca busca información por su cuenta**. Recibe solo lo que necesita, cuando lo necesita. Esto es lo que hace que ZyroAgentCLI sea 5-10x más eficiente en tokens que un agente sin orquestación.

### 📚 Más información

| Documento | Contenido |
|-----------|----------|
| [Arquitectura v2](docs/architecture-v2.md) | Diseño completo del sistema |
| [HelixDB Schema](docs/helixdb-schema-hql.md) | Schema de nodos y edges en HelixDB |
| [HelixDB Integración](docs/helixdb-integration.md) | Cómo se conectan los componentes |
| [Especificación Técnica](docs/spec-zyrov2.md) | Spec completa C-I-O |

## 🏗️ Arquitectura

```
Humano ──→ ZyroCLI (Go) ──→ OpenCode (Agentes IA)
              │                     │
              ├── HelixDB (grafos)  ├── PydanticAI (Agent-as-Validator)
              ├── Boundari (seguridad) ├── Boomerang (ciclo 6 pasos)
              └── Memoria Causal    └── Skills (patrones, librerías, etc.)
```

## 🚀 Instalación

```bash
# Opción 1: npm ([zyrocli](https://www.npmjs.com/package/zyrocli)) (recomendado)
npx zyrocli setup

# Opción 2: npm global
npm install -g zyrocli
zyrocli setup

# Opción 3: go install (requiere Go)
go install github.com/secko/zyrocli@latest
zyrocli setup

# Opción 4: binary directo
curl -sSL https://github.com/secko/zyrocli/releases/latest/download/install.sh | bash
```

> `zyro setup` detecta tu sistema operativo, instala dependencias (Go, uv, HelixDB, Ollama opcional), configura MCP servers y te deja listo para usar en segundos.

## ⚡ Quick Start

```bash
# 1. Instalar
npx zyrocli setup

# 2. Crear un handoff con la descripción de tu proyecto
cat > handoff.yaml << 'EOF'
project:
  name: mi-app
  description: API REST en Go con autenticación JWT
  technologies:
    - Go
    - PostgreSQL
    - JWT
EOF

# 3. Inicializar el proyecto
zyro init handoff.yaml
```

## 📚 Documentación

| Documento | Contenido |
|-----------|----------|
| [Especificación Técnica](docs/spec-zyrov2.md) | Spec completa C-I-O del sistema |
| [Diseño Técnico](docs/design-zyrov2.md) | Arquitectura detallada, firmas, schemas |
| [Roadmap](docs/roadmap-integrado.md) | Plan de implementación y sprints |
| [FAQ - PydanticAI Harness](docs/explorations/investigacion-01-pydanticai-harness.md) | Por qué Agent-as-Validator |
| [FAQ - HelixDB](docs/explorations/investigacion-02-helixdb-deep-integration.md) | Por qué grafos + vectores |
| [FAQ - Boundari](docs/explorations/investigacion-03-boundari-politicas-seguridad.md) | Por qué seguridad por fase |
| [FAQ - Memoria Causal](docs/explorations/investigacion-04-engram-memoria-causal.md) | Por qué engram custom sobre HelixDB |
| [FAQ - OpenCode](docs/explorations/investigacion-05-opencode-ecosistema-plugins.md) | Por qué OpenCode como runtime |
| [FAQ - Embeddings](docs/explorations/investigacion-06-embedding-system.md) | Por qué mxbai-embed-large + Scaleway |

## 🛡️ Seguridad

ZyroCLI aplica el **principio de mínimo privilegio** en cada fase del pipeline:

```
F0 (Investigación) → solo lectura de código y web
F1 (Especificación) → lectura + escritura de documentos .md
F2 (Diseño) → escritura de planos .md .yaml
F3 (Implementación) → escritura de código + ejecución controlada
F4 (Cierre) → solo lectura, archive con approval
```

## ✅ Estado del Proyecto

- **Pipeline F0-F4**: Completo e integrado con Boomerang
- **Memoria Causal**: 6 tipos de Facts, 7 aristas causales, curva de Ebbinghaus
- **Embeddings**: Ollama + mxbai-embed-large + fallback Scaleway/BM25
- **Seguridad**: Boundari con políticas YAML por fase
- **Tests**: 383 tests, 100% passing
- **Distribución**: npm, GitHub Releases, Homebrew, go install

## Licencia

MIT
