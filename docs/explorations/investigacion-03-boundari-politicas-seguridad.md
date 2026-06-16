# Investigación: Boundari como Capa de Seguridad/Políticas para Agentes Python

> **Fecha:** 2026-06-15
> **Exploración:** Fase 0 — Investigación técnica
> **Contexto:** ZyroAgentCLI necesita controlar qué herramientas (MCP tools) pueden usar los agentes Python en cada fase del pipeline SDD.
> **Estado actual:** Sin restricciones — el agente puede llamar a cualquier tool. Se busca Boundari para implementar políticas YAML por fase.

---

## Resumen Ejecutivo

**Boundari** es una librería Python (v0.1.0, MIT, por Tayor) que implementa **políticas-as-código** para agentes de IA. Permite definir en YAML qué tools están permitidas/denegadas, con gates de aprobación humana, budgets, validación de schemas, redacción de datos sensibles, y auditoría — todo sin forzar un framework de agentes específico.

**Hallazgo clave para ZyroAgentCLI:**
- Boundari **funciona perfectamente** para controlar MCP tools en agentes Python.
- Soporta políticas separadas por fase (cada fase carga su propio `boundari.yaml`).
- El decorador `@boundary_tool` y el wrapper `boundary.wrap_tool()` permiten integrar sin modificar la lógica del agente.
- La versión es **0.1.0 (alpha)** — madurez temprana pero funcionalidad sólida.

**Limitaciones detectadas:**
- El proyecto es muy nuevo (1 release, 0 stars, 0 forks en GitHub).
- No soporta políticas diferenciales por *fase* de forma nativa — hay que instanciar múltiples `Boundary` objects.
- Las políticas de red (network egress) y filesystem no están incorporadas directamente — se modelan como tools.
- Los adapters para Pydantic AI y OpenAI Agents SDK son wrappers mínimos.

---

## ¿Qué es Boundari y Cómo Funciona?

Boundari se describe como **"Policy-as-code runtime boundaries for Python AI agents"**. Funciona como una **capa de enforcement** entre el agente y las tools que invoca:

```
Agente → decide llamar a tool X
              ↓
         [Boundary.check()]
              ↓
    ┌─────────┴──────────┐
    │  ¿Tool permitida?   │── No → Decision(allowed=False)
    │  ¿Budget disponible?│── No → Decision(allowed=False)
    │  ¿Requiere approval?│── Sí → ApprovalRequest → ¿Aprobado?
    │  ¿Schema válido?    │── No → Decision(allowed=False)
    └─────────┬──────────┘
              ↓ Sí
         [Redactor.redact()]
              ↓
         Ejecuta tool real → output redactado → devuelve al agente
```

### Core Concepts (desde el código fuente)

| Concepto | Archivo | Descripción |
|----------|---------|-------------|
| `Boundary` | `boundary.py` | Contrato runtime para un agente/workflow. Contiene tools, budget, approver, redactor, auditor. |
| `ToolPolicy` | `policy.py` | Política para una tool: nombre, allowed, approval_required, approval_condition, risk, scopes, schemas, allowed_tables, max_amount. |
| `Decision` | `boundary.py` | Resultado estructurado: `allowed: bool`, `tool_name`, `reason`, `message`, `requires_approval`, `metadata`. |
| `Budget` | `budget.py` | Límites: `max_tool_calls`, `max_runtime_seconds`, `max_cost_usd`, `max_tokens`. |
| `RunContext` | `budget.py` | Estado mutable de una ejecución: `run_id`, `started_at`, `tool_calls`, `cost_usd`, `tokens`. |
| `Redactor` | `redact.py` | Enmascara strings sensibles (API keys, emails, tarjetas, teléfonos) en outputs. |
| `AuditEvent` / `AuditSink` | `audit.py` | Eventos estructurados de allow/deny/approval. Sinks: `MemoryAuditLog`, `JSONLAuditLog`. |
| `ApprovalRequest` | `approval.py` | Info presentada al aprobador humano: tool_name, args_summary, risk, reason. |

### Flujo Interno Detallado

Cuando se llama a una tool envuelta (`boundary.wrap_tool`):

1. **Input validation** (`_validate_input`): si la policy tiene `input_model`, valida args contra el schema Pydantic.
2. **Precheck** (`_precheck`):
   - ¿Tool existe y `allowed=True`? Si no → `tool_not_allowed`
   - Si `outputs_require_schema=True` y tool no tiene output_schema → `output_schema_required`
   - ¿Budget disponible? → `budget_exceeded`
   - ¿Tabla permitida? (SQL table scoping) → `table_not_allowed`
   - ¿Monto excede límite? → `amount_exceeds_limit`
   - ¿Requiere approval? → `approval_required` → `ApprovalRequest`
3. **Approval** (si requiere): callback sincrónico/async → `ApprovalResult`
4. **Ejecución**: si todo pasa, ejecuta la tool real
5. **Output validation**: valida contra `output_model` si existe
6. **Blocked output check**: si output contiene strings en `block_if_contains`
7. **Redacción**: `Redactor.redact_value()` sobre el resultado antes de devolverlo

---

## Sintaxis Completa de `boundari.yaml`

### Generación

```bash
pip install boundari
boundari init          # Crea boundari.yaml con política ejemplo
boundari validate      # Valida sintaxis y reglas estáticas
boundari test          # Ejecuta pruebas contra golden traces
```

### Estructura Completa

```yaml
# Identificador del agente
agent: nombre_del_agente        # Opcional. Alternativa: name
name: support_agent             # Alternativa a agent

# ---- Budgets ----
budgets:
  max_tool_calls: 20           # Máximo de llamadas a tools por run
  max_runtime_seconds: 180     # Tiempo máximo en segundos
  max_cost_usd: "0.50"         # Costo máximo en USD (string para precisión Decimal)
  max_tokens: 10000            # Máximo de tokens

# ---- Políticas de Tools ----
tools:
  # Tool permitida sin restricciones
  docs.search:
    allow: true
    scopes: ["docs:read"]
    risk: low
    output_schema: SearchResult

  # Tool con approval condicional
  email.send:
    allow: true
    require_approval:
      when: "recipient_domain not in trusted_domains"   # Expresión evaluada con safe_eval
    scopes: ["email:send"]
    risk: high               # low | medium | high
    input_schema: EmailInput     # Nombre del schema Pydantic
    output_schema: EmailResult

  # Tool con approval siempre
  stripe.refund:
    allow: true
    require_approval: true
    scopes: ["stripe:refund"]
    risk: high
    max_amount: "100.00"         # Límite de monto en USD
    output_schema: RefundResult

  # Tool con scoping de tablas SQL
  db.query:
    allow: true
    allowed_tables:
      - customers
      - orders
    scopes: ["db:read"]

  # Tool denegada explícitamente
  shell.run:
    allow: false

  # Tool con schema de entrada
  filesystem.write:
    allow: true
    risk: high
    input_schema: FileWriteInput
    output_schema: FileWriteResult

# ---- Configuración de Datos ----
data:
  redact:
    - api_key            # Patrón: sk-xxx, api_key=xxx
    - credit_card        # Patrón: 16 dígitos
    - email              # Patrón: user@domain
    - phone              # Patrón: números telefónicos
  trusted_domains:
    - example.com
    - empresa.cl

# ---- Control de Outputs ----
outputs:
  require_schema: true          # Exige output_schema en todas las tools permitidas
  block_if_contains:            # Bloquea output si contiene estos strings
    - "{{SECRET}}"
    - "PASSWORD"

# ---- Pruebas de Política ----
policy_tests:
  forbidden_tools:             # Tools que NUNCA deben estar permitidas
    - shell.run
    - filesystem.delete
  golden_traces:               # Archivos JSONL para replay testing
    - traces/test_run.jsonl
    - traces/production.jsonl
```

### Sintaxis de `require_approval.when`

La expresión `when` se evalúa con un **safe evaluator** (AST-based, no `eval()` peligroso). Variables disponibles:

- `recipient_domain`: extraída de args `to`, `recipient`, o `email`
- `trusted_domains`: lista del YAML
- + todos los argumentos posicionales/nombre de la tool

Operadores soportados: `==`, `!=`, `in`, `not in`, `<`, `<=`, `>`, `>=`, `and`, `or`, `not`.

Ejemplo:
```yaml
require_approval:
  when: "recipient_domain not in trusted_domains and amount > '50.00'"
```

---

## Integración con Python

### 1. API Directa (ToolPolicy objects)

```python
from decimal import Decimal
from pydantic import BaseModel, EmailStr
from boundari import Boundary, Budget, ToolPolicy

# Schemas Pydantic para validación
class EmailInput(BaseModel):
    to: EmailStr
    subject: str
    body: str

class EmailResult(BaseModel):
    message_id: str
    status: str
    to: EmailStr

# Crear boundary programáticamente
boundary = Boundary(
    name="mi_agente",
    budget=Budget(
        max_tool_calls=20,
        max_runtime_seconds=180,
        max_cost_usd=Decimal("0.50"),
        max_tokens=10000,
    ),
    tools=[
        ToolPolicy("docs.search").allow().with_scopes(["docs:read"]),
        ToolPolicy("email.send")
            .require_approval(when="recipient_domain not in trusted_domains")
            .input(EmailInput)
            .output(EmailResult)
            .with_risk("high"),
        ToolPolicy("shell.run").deny(),
        ToolPolicy("stripe.refund")
            .require_approval()
            .max_amount("100.00"),
        ToolPolicy("db.query")
            .allow()
            .allow_tables(["customers", "orders"]),
    ],
    trusted_domains=["example.com"],
    outputs_require_schema=True,
    block_if_contains=["{{SECRET}}"],
)
```

### 2. Desde YAML

```python
from boundari import Boundary, console_approver

# Cargar política desde archivo YAML
boundary = Boundary.from_file("boundari.yaml")

# Con approver humano (consola)
boundary = Boundary.from_file("boundari.yaml", approver=console_approver)

# Con approver personalizado
def my_approver(request):
    # Lógica: llamar a API, Slack, etc.
    return request.tool_name != "stripe.refund"

boundary = Boundary.from_file("boundari.yaml", approver=my_approver)
```

### 3. Wrapper de Tools (wrap_tool)

```python
# --- Tool real ---
def send_email(to: str, subject: str, body: str) -> dict:
    # ... lógica real de envío ...
    return {"message_id": "msg_123", "status": "sent", "to": to}

# Envolver con boundary
safe_send_email = boundary.wrap_tool("email.send", send_email)

# Usar la tool envuelta
result = safe_send_email(
    to="customer@outside.test",
    subject="Refund update",
    body="We can help with that.",
)

# result es un Decision o el resultado real
if isinstance(result, dict) and "message_id" in result:
    print("Email sent:", result)
else:
    print(f"Bloqueado: {result.reason}")  # "approval_denied"

# Con raise_on_denied para que lance excepción
safe_send_email_strict = boundary.wrap_tool(
    "email.send", send_email, raise_on_denied=True
)
```

### 4. Decorator API

```python
from boundari import boundary_tool

# Decorador: adjunta metadata Boundari a la función
@boundary_tool(
    name="email.send",
    risk="high",
    scopes=["email:send"],
    require_approval=True,
)
async def send_email(to: str, subject: str, body: str) -> dict:
    return {"message_id": "msg_123", "status": "queued"}

# Acceder a la policy desde la función
policy = send_email.__boundari_tool_policy__
```

### 5. Con RunContext (budget por ejecución)

```python
from boundari import RunContext

# Crear un contexto para una ejecución específica
context = boundary.new_run_context(run_id="run_001")

# Usar el contexto en el wrapper
safe_tool = boundary.wrap_tool("docs.search", search_func, context=context)

# Ver estado del budget
print(f"Tool calls: {context.tool_calls}")
print(f"Runtime: {context.runtime_seconds}s")
```

### 6. Con Framework Adapters (Pydantic AI / OpenAI Agents SDK)

```python
# Pydantic AI
from boundari import Boundary
from boundari.adapters.pydantic_ai import wrap_agent

agent = Agent(...)  # tu agente Pydantic AI
safe_agent = wrap_agent(agent, boundary=Boundary.from_file("boundari.yaml"))

# OpenAI Agents SDK
from boundari.adapters.openai_agents import wrap_agent

agent = Agent(...)  # tu agente OpenAI SDK
safe_agent = wrap_agent(agent, boundary=Boundary.from_file("boundari.yaml"))
```

### 7. Auditoría

```python
from boundari import Boundary, JSONLAuditLog

# Auditoría a archivo JSONL
auditor = JSONLAuditLog("auditoria/mi_agente.jsonl")
boundary = Boundary.from_file("boundari.yaml", auditor=auditor)

# En memoria (default)
from boundari import MemoryAuditLog
memory_auditor = MemoryAuditLog()
boundary = Boundary.from_file("boundari.yaml", auditor=memory_auditor)
```

### 8. Redacción Personalizada

```python
from boundari import Redactor, RedactionRule
import re

# Reglas por defecto: api_key, credit_card, email, phone
redactor = Redactor(["email", "api_key"])

# Reglas personalizadas
custom_redactor = Redactor(
    rules=["api_key", "email"],
    custom_patterns={
        "rut": r"\b\d{1,2}\.\d{3}\.\d{3}[-][0-9Kk]\b",
        "token_interno": r"tkn_[a-z0-9]{32}",
    }
)

# Usar en boundary
boundary = Boundary(name="mi_agente", redactor=custom_redactor)
```

### 9. Testing con Golden Traces

```bash
# Replay: evaluar decisions sin ejecutar tools
boundari replay traces/run_001.jsonl --policy boundari.yaml

# Explicar resultados de un trace
boundari explain traces/run_001.jsonl

# Validación CI
boundari test boundari.yaml
```

Formato JSONL para traces:

```json
{"tool": "docs.search", "args": {"query": "refund policy"}, "expected_allowed": true}
{"tool": "email.send", "args": {"to": "outside@evil.com", "subject": "test", "body": "test"}, "expected_allowed": false}
{"tool": "shell.run", "args": {"command": "rm -rf /"}, "expected_allowed": false}
```

---

## Políticas Recomendadas para Cada Fase (F0-F4)

### Modelo: cada fase carga su propio `boundari.yaml`

La idea es tener un archivo de política por fase, y el scheduler de ZyroAgentCLI selecciona cuál aplicar según la fase actual:

```
~/.config/zyroagent/policies/
├── phase0-boundari.yaml    # Descubrimiento: solo lectura
├── phase1-boundari.yaml    # Investigación: solo lectura + approval
├── phase2-boundari.yaml    # Plan: lectura + escritura con approval fuerte
├── phase3-boundari.yaml    # Implementación: lectura/escritura con gates
└── phase4-boundari.yaml    # Review: solo lectura otra vez
```

### F0 — Descubrimiento (Discovery)

**Característica:** Solo lectura del código base y documentación. Sin escritura, sin ejecución.

```yaml
# phase0-boundari.yaml
name: phase0_discovery

budgets:
  max_tool_calls: 30
  max_runtime_seconds: 300
  max_cost_usd: "0.10"

tools:
  # --- Tools de lectura permitidas ---
  read_file:
    allow: true
    scopes: ["fs:read"]
    risk: low

  search_code:
    allow: true
    scopes: ["code:search"]
    risk: low

  list_directory:
    allow: true
    scopes: ["fs:read"]
    risk: low

  git_log:
    allow: true
    scopes: ["git:read"]
    risk: low

  git_diff:
    allow: true
    scopes: ["git:read"]
    risk: low

  # --- Tools de sistema bloqueadas ---
  write_file:
    allow: false

  delete_file:
    allow: false

  shell_exec:
    allow: false

  network_request:
    allow: false

  npm_install:
    allow: false

  git_commit:
    allow: false

  git_push:
    allow: false

outputs:
  require_schema: false
  block_if_contains:
    - "{{SECRET}}"
    - "PASSWORD"
    - "api_key"
    - "sk-"

policy_tests:
  forbidden_tools:
    - write_file
    - delete_file
    - shell_exec
    - git_commit
    - git_push
```

### F1 — Investigación (Investigation)

**Característica:** Lectura intensiva + herramientas de investigación. Approval requerido para cualquier cosa que toque APIs externas.

```yaml
# phase1-boundari.yaml
name: phase1_investigation

budgets:
  max_tool_calls: 40
  max_runtime_seconds: 600
  max_cost_usd: "0.25"

tools:
  # Lectura intensiva
  read_file:
    allow: true
    risk: low

  search_code:
    allow: true
    risk: low

  grep_search:
    allow: true
    risk: low

  list_directory:
    allow: true
    risk: low

  # Investigación externa
  web_fetch:
    allow: true
    require_approval:
      when: "'localhost' not in url and '127.0.0.1' not in url"
    scopes: ["web:fetch"]
    risk: medium

  pypi_search:
    allow: true
    scopes: ["pkg:search"]
    risk: low

  github_search:
    allow: true
    require_approval:
      when: "repo not in trusted_repos"
    risk: medium

  # Tools bloqueadas
  write_file:
    allow: false

  shell_exec:
    allow: false

  network_request:
    allow: false

data:
  redact:
    - api_key
    - email
    - credit_card
    - phone
  trusted_domains:
    - pypi.org
    - github.com
    - docs.python.org

outputs:
  block_if_contains:
    - "{{SECRET}}"

policy_tests:
  forbidden_tools:
    - shell_exec
    - write_file
```

### F2 — Planificación (Planning)

**Característica:** Escritura de planos/documentos. Approval requerido para escribir fuera de directorios planificados.

```yaml
# phase2-boundari.yaml
name: phase2_planning

budgets:
  max_tool_calls: 50
  max_runtime_seconds: 600
  max_cost_usd: "0.35"

tools:
  # Lectura
  read_file:
    allow: true
    risk: low

  # Escritura de documentos de plan
  write_file:
    allow: true
    require_approval: true         # Toda escritura requiere aprobación
    scopes: ["plan:write"]
    risk: high

  create_directory:
    allow: true
    require_approval: true
    risk: high

  # Shell solo para comandos seguros
  shell_exec:
    allow: true
    require_approval:
      when: "command not in safe_commands"
    scopes: ["shell:read"]
    risk: high

  # Tools bloqueadas
  delete_file:
    allow: false

  npm_install:
    allow: false

  git_commit:
    allow: true
    require_approval: true
    risk: high

  git_push:
    allow: false

data:
  safe_commands:
    - "ls"
    - "pwd"
    - "cat"
    - "which"
    - "echo"
    - "mkdir -p"

  redact:
    - api_key
    - email
    - credit_card

outputs:
  require_schema: true
  block_if_contains:
    - "{{SECRET}}"

policy_tests:
  forbidden_tools:
    - delete_file
    - git_push
```

### F3 — Implementación (Implementation)

**Característica:** Escritura intensiva. Gates de approval para operaciones destructivas. Budget más generoso.

```yaml
# phase3-boundari.yaml
name: phase3_implementation

budgets:
  max_tool_calls: 150
  max_runtime_seconds: 1800
  max_cost_usd: "1.00"
  max_tokens: 50000

tools:
  # Lectura/escritura intensiva de código
  read_file:
    allow: true
    risk: low

  write_file:
    allow: true
    require_approval:
      when: "'src/' not in path and 'lib/' not in path"    # Solo approval si no es en src/
    scopes: ["code:write"]
    risk: high

  create_directory:
    allow: true
    risk: low

  # Shell para comandos de build/test
  shell_exec:
    allow: true
    require_approval: true
    scopes: ["shell:exec"]
    risk: high

  # Paquetes
  npm_install:
    allow: true
    require_approval:
      when: "package_count > 3"
    risk: high

  pip_install:
    allow: true
    require_approval: true
    risk: high

  # Git
  git_commit:
    allow: false

  git_push:
    allow: false

  git_diff:
    allow: true
    risk: low

  delete_file:
    allow: false

data:
  redact:
    - api_key
    - email
    - credit_card

outputs:
  require_schema: false
  block_if_contains:
    - "{{SECRET}}"

policy_tests:
  forbidden_tools:
    - delete_file
    - git_commit
    - git_push
```

### F4 — Revisión (Review)

**Característica:** Modo solo lectura otra vez. Approval requerido solo si se necesita corregir algo.

```yaml
# phase4-boundari.yaml
name: phase4_review

budgets:
  max_tool_calls: 30
  max_runtime_seconds: 300
  max_cost_usd: "0.10"

tools:
  # Solo lectura por defecto
  read_file:
    allow: true
    risk: low

  search_code:
    allow: true
    risk: low

  git_log:
    allow: true
    risk: low

  git_diff:
    allow: true
    risk: low

  list_directory:
    allow: true
    risk: low

  # Escritura permitida solo bajo approval
  write_file:
    allow: true
    require_approval: true
    risk: high
    scopes: ["code:write"]

  shell_exec:
    allow: false

  delete_file:
    allow: false

  npm_install:
    allow: false

  pip_install:
    allow: false

  network_request:
    allow: false

  git_commit:
    allow: true
    require_approval: true
    risk: high

  git_push:
    allow: true
    require_approval: true
    risk: high

policy_tests:
  forbidden_tools:
    - delete_file
    - shell_exec
    - npm_install
    - pip_install
```

### Cargar Política por Fase en el Scheduler

```python
from pathlib import Path
from boundari import Boundary, console_approver

BOUNDARI_DIR = Path.home() / ".config" / "zyroagent" / "policies"

POLICIES = {
    "phase0": "phase0-boundari.yaml",
    "phase1": "phase1-boundari.yaml",
    "phase2": "phase2-boundari.yaml",
    "phase3": "phase3-boundari.yaml",
    "phase4": "phase4-boundari.yaml",
}

def get_boundary(phase: str) -> Boundary:
    """Obtener Boundary para la fase dada."""
    filename = POLICIES.get(phase, "phase0-boundari.yaml")
    path = BOUNDARI_DIR / filename
    return Boundary.from_file(str(path), approver=console_approver)

# Uso en el scheduler
def run_agent(phase: str, agent_func, **tool_args):
    boundary = get_boundary(phase)
    safe_func = boundary.wrap_tool("mi_tool", agent_func)
    return safe_func(**tool_args)
```

---

## Alternativas a Boundari

Boltzmann no sería suficiente si Boundari no cubre ciertos casos. Aquí las alternativas:

### 1. Guardrails AI (guardrails) — v2.0.0

| Aspecto | Boundari | Guardrails AI |
|---------|----------|---------------|
| Enfoque | Políticas de tools a nivel runtime | Validación de output del LLM |
| YAML | Sí, nativo | Sí, RAIL spec |
| Tool gating | Sí (allow/deny/approval) | No directamente |
| Schema validation | Sí (Pydantic) | Sí (Pydantic) |
| Budgets | Sí | No |
| Redacción | Sí | No |
| Audit | Sí (JSONL/Memory) | No |
| Madurez | v0.1.0 alpha | v2.0.0 estable |
| Ideal para | **Control de tools de agentes** | Validación de respuestas de LLMs |

**Veredicto:** No reemplaza a Boundari. Guardrails se enfoca en lo que el LLM *dice*, no en lo que el agente *hace*.

### 2. Políticas en OpenCode (opencode.jsonc)

OpenCode ya tiene un sistema de políticas basado en archivos JSON/JSONC. Se pueden definir:

```jsonc
// ~/.config/opencode/opencode.jsonc
{
  "mcp": {
    "tools": {
      "read_file": { "allow": true },
      "write_file": { "allow": false },
      "shell_exec": { "allow": false }
    },
    "budgets": {
      "max_tool_calls_per_run": 50,
      "max_runtime_seconds": 300
    }
  }
}
```

**Limitación:** Las políticas de OpenCode son globales — no hay soporte nativo para políticas *por fase* o *por agente*. Un MCP tool server puede implementar su propio gating.

**Veredicto:** Útil como complemento, no como reemplazo para políticas por fase.

### 3. MCP Gates / MCP Authorization Layer

El protocolo MCP no tiene autorización incorporada, pero se puede implementar:

- **MCP middleware**: un proxy que intercepta requests MCP y aplica políticas antes de reenviarlas al tool server.
- **MCP tool server wrapper**: cada tool server implementa su propia capa de autorización.

```python
# Ejemplo: proxy MCP con Boundari
from boundari import Boundary

class MCPPolicyMiddleware:
    """Middleware que aplica Boundari a requests MCP."""
    
    def __init__(self, boundary: Boundary, upstream_url: str):
        self.boundary = boundary
        self.upstream = upstream_url
    
    async def handle_request(self, tool_name: str, args: dict) -> dict:
        decision = self.boundary.decide(tool_name, args)
        if not decision.allowed:
            return {"error": "denied", "reason": decision.reason}
        
        # Reenviar al tool server
        return await self._forward(tool_name, args)
```

**Veredicto:** Opción válida si se necesita el enforcement a nivel de protocolo MCP en vez de a nivel de agente Python.

### 4. In-house Policy Engine (Go-based)

Dado que ZyroAgentCLI está escrito en Go, se podría implementar un policy engine nativo:

```go
type ToolPolicy struct {
    Name             string
    Allowed          bool
    RequireApproval  bool
    AllowedTables    []string
    MaxAmount        *decimal.Decimal
}

type Boundary struct {
    Name    string
    Tools   map[string]ToolPolicy
    Budget  Budget
}

func (b *Boundary) Decide(toolName string, args map[string]interface{}) Decision {
    policy, ok := b.Tools[toolName]
    if !ok || !policy.Allowed {
        return Decision{Allowed: false, Reason: "tool_not_allowed"}
    }
    // ... checks
}
```

**Ventaja:** Sin dependencia Python, ejecuta en el mismo proceso Go.
**Desventaja:** Duplica funcionalidad de Boundari. Solo justificable si no se usan agentes Python.

### 5. Tabla Comparativa

| Feature | Boundari | Guardrails AI | OpenCode Policies | MCP Gates | In-house Go |
|---------|----------|---------------|-------------------|-----------|-------------|
| Tool allow/deny | ✅ | ❌ | ⚠️ | ✅ | ✅ |
| Approval gates | ✅ | ❌ | ❌ | ⚠️ | ✅ |
| Budgets | ✅ | ❌ | ❌ | ❌ | ✅ |
| Schema validation | ✅ | ✅ | ❌ | ❌ | ✅ |
| Redacción | ✅ | ❌ | ❌ | ❌ | ❌ |
| Auditoría | ✅ | ❌ | ❌ | ✅ | ✅ |
| Por fase | ✅ | ❌ | ❌ | ✅ | ✅ |
| YAML nativo | ✅ | ✅ | JSONC | ❌ | ❌ |
| Madurez | α | ✅ | ✅ | ❌ | ❌ |
| Multi-framework | ✅ | ✅ | Solo OpenCode | Cualquier MCP | Solo Go |
| Dependencia | Python | Python | OpenCode | Python/JS/Go | Go |

---

## Recomendaciones Concretas para ZyroAgentCLI

### Decisión: **SÍ integrar Boundari**

Boundari es la mejor opción para el control de políticas de agentes Python en ZyroAgentCLI por las siguientes razones:

### 1. Arquitectura Propuesta

```
ZyroAgentCLI (Go)
    │
    ├── Scheduler (Go)
    │     └── Decide qué fase ejecutar (F0-F4)
    │
    ├── Políticas YAML por fase
    │     └── ~/.config/zyroagent/policies/phase{N}-boundari.yaml
    │
    └── Agentes Python (proceso hijo o subproceso)
          │
          ├── 👉 Carga Boundary desde archivo YAML de la fase actual
          ├── 👉 Envuelve cada MCP tool con `boundary.wrap_tool()`
          ├── 👉 Ejecuta sin riesgo de llamadas no controladas
          └── 👉 Después de cada run, el scheduler puede inspeccionar auditoría
```

### 2. Flujo de Integración

```python
# pseudo-código Python para el agente ZyroAgentCLI
import sys
import json
from boundari import Boundary

def main():
    # El scheduler Go pasa la fase como argumento
    phase = sys.argv[1]  # "phase0", "phase1", etc.
    
    # Cargar política de la fase
    boundary = Boundary.from_file(f"policies/{phase}-boundari.yaml")
    
    # Envolver todas las MCP tools
    for tool_name, tool_func in mcp_tools_registry.items():
        mcp_tools_registry[tool_name] = boundary.wrap_tool(
            tool_name, tool_func,
            raise_on_denied=True  # o False para Decision objects
        )
    
    # Ejecutar el agente (sus llamadas pasan por el boundary)
    agent.run()
```

### 3. Integración con el Scheduler Go

El scheduler Go en `internal/scheduler/` puede:

1. **Antes del agente:** Escribir temporalmente la política YAML correspondiente a la fase
2. **Pasar la fase:** Como variable de entorno o argumento al proceso Python
3. **Después del agente:** Leer el archivo de auditoría JSONL para registrar qué tools se llamaron
4. **Approval gates:** El scheduler Go intercepta decisiones de approval a través de `console_approver` o un callback HTTP vía `fastapi_approval_router`

### 4. Pasos para Implementar

| # | Acción | Detalle |
|---|--------|---------|
| 1 | `pip install boundari` | En el entorno Python del agente |
| 2 | Crear `policies/` | 5 archivos YAML (F0-F4) según las plantillas de arriba |
| 3 | Modificar `agent.py` | Agregar `Boundary.from_file()` antes del loop principal del agente |
| 4 | Envolver MCP tools | `boundary.wrap_tool()` en cada tool registrada |
| 5 | Integrar approval | Usar `console_approver` para desarrollo, `fastapi_approval_router` para producción |
| 6 | Auditoría | Configurar `JSONLAuditLog` por fase |
| 7 | Tests | Crear golden traces para políticas críticas |
| 8 | CI pipeline | Agregar `boundari test` en el CI |

### 5. Riesgos y Mitigaciones

| Riesgo | Mitigación |
|--------|------------|
| Boundari v0.1.0 alpha — cambios API | Congelar versión en requirements.txt, hacer pinning exacto |
| Python dependency en proyecto Go | Ya hay agentes Python, no es nueva dependencia |
| Safe eval limitado | Usar approval_condition simple; no depender de expresiones complejas |
| Sin soporte de network policies | Modelar network como tools con allow/deny como cualquier otra tool |
| Sin soporte filesystem paths | Modelar filesystem access como tools con approval condicional |

### 6. Costo de Integración

- **Dependencias nuevas:** 1 (boundari) + sus dependencias (pydantic, pyyaml, rich, typer) — ya ligeras
- **Líneas de código a agregar:** ~30-50 líneas en el agente Python wrapper
- **Archivos YAML:** 5 archivos de política (~40-60 líneas cada uno)
- **Pruebas:** golden traces por fase (~10-20 eventos cada una)
- **Impacto en rendimiento:** Mínimo — las comprobaciones son en memoria, sin I/O de red

### 7. Conclusión

Boundari es la solución correcta para el problema de ZyroAgentCLI. Su modelo de políticas YAML, el enforcement por tool, los approval gates humanos, y el sistema de budgets cubren exactamente las necesidades de control por fase. La integración es limpia (wrap_tool), no intrusiva (no requiere cambiar el framework del agente), y extensible (soporta schemas, redacción, auditoría).

**Recomendación:** Implementar Boundari como la capa de políticas estándar para todos los agentes Python en ZyroAgentCLI, con 5 archivos YAML (F0-F4) cargados dinámicamente según la fase activa.
