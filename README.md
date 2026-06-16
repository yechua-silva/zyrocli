# ZyroCLI — Orquestador Autónomo para Desarrollo Asistido por IA

> **Pipeline SDD completo (F0→F4) · Memoria Causal · Seguridad por Fase · Auto-instalable**

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
# Opción 1: npm (recomendado)
npx zyrocli setup

# Opción 2: npm global
npm install -g zyrocli
zyrocli setup

# Opción 3: go install
go install github.com/secko/zyrocli@latest
zyrocli setup

# Opción 4: script directo
curl -sSL https://github.com/secko/zyrocli/releases/latest/download/install.sh | bash
```

> `zyro setup` detecta tu sistema, instala dependencias (Go, uv, HelixDB, Ollama opcional), configura todo y te deja listo para usar.

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
