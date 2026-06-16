"""PydanticAI agent — Agent-as-Validator pattern.

The agent NEVER writes to HelixDB. It returns a validated PydanticModel.
The Go orchestrator handles all writes and side effects.

Read-only tools:
  - search_code(ctx, input) -> list[HelixSearchResult]
  - search_skills(ctx, input) -> list[HelixSearchResult]
  - task_context(ctx, task_id) -> str
  - save_to_helix(ctx, ...) -> RuntimeError (deprecated — never call)
"""

from __future__ import annotations

import json

from pydantic_ai import Agent, RunContext

from boundari_wrapper import BoundariWrapper
from capabilities import AgentDependencies
from helix_client import HelixClient
from models import AgentDecision, HelixReadInput, HelixSearchResult

# System prompt con reglas estrictas
SYSTEM_PROMPT = """Eres un agente validador de Zyro. Tu función es ANALIZAR y DECIDIR, nunca ejecutar.

REGLAS:
1. NUNCA intentes escribir en HelixDB — tu output será validado y escrito por el orquestador Go
2. NUNCA solicites tools de escritura — solo tienes acceso a herramientas de lectura
3. Si detectas contradicción con la memoria causal recibida, inclúyelo en tu razonamiento
4. Tu output debe ser un AgentDecision con:
   - action: qué acción recomiendas (search, create, update, skip)
   - reasoning: justificación detallada (mínimo 10 caracteres)
   - nodes: lista de nodos HelixNodeOutput sugeridos
   - requires_approval: True si la acción necesita aprobación humana
5. La fase actual y el contexto de memoria están disponibles en las dependencias
"""

zyro_agent: Agent[AgentDependencies, AgentDecision] = Agent(
    model="openai:gpt-5.2",
    system_prompt=SYSTEM_PROMPT,
    output_type=AgentDecision,
    deps_type=AgentDependencies,
)


# ---------------------------------------------------------------------------
# Read-only tools
# ---------------------------------------------------------------------------


@zyro_agent.tool
async def search_code(
    ctx: RunContext[AgentDependencies],
    input: HelixReadInput,
) -> list[HelixSearchResult]:
    """Busca código en HelixDB por texto. Solo lectura.

    Args:
        input: HelixReadInput con query texto y límite de resultados.

    Returns:
        Lista de HelixSearchResult con los IDs de los nodos encontrados.
    """
    client = HelixClient()
    try:
        nodes = await client.text_search("CodeNode", input.query, limit=input.limit)
        return [
            HelixSearchResult(
                id=n["id"],
                label="CodeNode",
                content=input.query,
                score=0.0,
                source="helixdb",
            )
            for n in nodes
        ]
    except Exception as exc:
        return [
            HelixSearchResult(
                id=0,
                label="Error",
                content=f"HelixDB search failed: {exc}",
                score=0.0,
                source="error",
            )
        ]


@zyro_agent.tool
async def search_skills(
    ctx: RunContext[AgentDependencies],
    input: HelixReadInput,
) -> list[HelixSearchResult]:
    """Busca skills en HelixDB por texto. Solo lectura.

    Args:
        input: HelixReadInput con query texto y límite de resultados.

    Returns:
        Lista de HelixSearchResult con los IDs de los skills encontrados.
    """
    client = HelixClient()
    try:
        nodes = await client.text_search("Skill", input.query, limit=input.limit)
        return [
            HelixSearchResult(
                id=n["id"],
                label="Skill",
                content=input.query,
                score=0.0,
                source="helixdb",
            )
            for n in nodes
        ]
    except Exception as exc:
        return [
            HelixSearchResult(
                id=0,
                label="Error",
                content=f"HelixDB search failed: {exc}",
                score=0.0,
                source="error",
            )
        ]


@zyro_agent.tool
async def task_context(
    ctx: RunContext[AgentDependencies],
    task_id: int,
) -> str:
    """Obtiene contexto completo de una tarea desde HelixDB.

    Devuelve un JSON estructurado con seis secciones:
    skills, code, docs, patterns, dependents, dependencies.

    Args:
        task_id: ID del nodo Task en HelixDB.

    Returns:
        String JSON con el contexto completo de la tarea.
    """
    client = HelixClient()

    # Fetch the task node first
    task_node = await client.get_node("Task", task_id)
    if task_node is None:
        return json.dumps(
            {"error": f"Task {task_id} not found", "helix_url": client.base_url},
            indent=2,
        )

    # Run six traversals
    sections: dict[str, list[dict]] = {
        "skills": [],
        "code": [],
        "docs": [],
        "patterns": [],
        "dependents": [],
        "dependencies": [],
    }

    try:
        sections["skills"] = await client.get_outgoing(task_id, "has_skill")
        sections["code"] = await client.get_outgoing(task_id, "has_code")
        sections["docs"] = await client.get_outgoing(task_id, "has_doc")
        sections["patterns"] = await client.get_outgoing(task_id, "has_pattern")
        sections["dependents"] = await client.get_incoming(task_id, "depends_on")
        sections["dependencies"] = await client.get_outgoing(task_id, "depends_on")
    except Exception as exc:
        return json.dumps(
            {
                "error": f"HelixDB query failed: {exc}",
                "helix_url": client.base_url,
                "task_id": task_id,
            },
            indent=2,
        )

    return json.dumps(
        {
            "task_id": task_id,
            "skills": sections["skills"],
            "code": sections["code"],
            "docs": sections["docs"],
            "patterns": sections["patterns"],
            "dependents": sections["dependents"],
            "dependencies": sections["dependencies"],
        },
        indent=2,
    )


@zyro_agent.tool(retries=0, requires_approval=True)
async def save_to_helix(
    ctx: RunContext[AgentDependencies],
    label: str,
    properties: dict,
) -> str:
    """DEPRECATED: No llamar. Los writes los maneja el orquestador Go.

    Args:
        label: ignored
        properties: ignored

    Raises:
        RuntimeError: siempre — el agente no puede escribir en HelixDB.
    """
    raise RuntimeError(
        "No puedes escribir en HelixDB. "
        "Los writes los hace el orquestador Go después de validar tu output."
    )


# ---------------------------------------------------------------------------
# Runner helper
# ---------------------------------------------------------------------------


async def run_agent(
    input_data: str,
    deps: AgentDependencies,
) -> AgentDecision:
    """Execute the agent with the given input and dependencies.

    Applies Boundari policy wrapper if a phase is specified in deps.
    Wraps each registered tool to enforce phase-specific boundaries
    (allow/deny/require_approval) and budget limits.
    """
    # ------------------------------------------------------------------
    # Apply Boundari wrapper if phase is specified
    # ------------------------------------------------------------------
    boundari_wrapper: BoundariWrapper | None = None
    if deps.boundari_phase:
        try:
            boundari_wrapper = BoundariWrapper(phase=deps.boundari_phase)
            for tool in zyro_agent._function_toolset.tools.values():
                original_fn = tool.function
                wrapped = boundari_wrapper.wrap_tool(tool.name, original_fn)
                tool.function = wrapped
        except Exception:
            pass  # Fallback: continue without Boundari

    result = await zyro_agent.run(input_data, deps=deps)

    # ------------------------------------------------------------------
    # Save audit log post-execution
    # ------------------------------------------------------------------
    if boundari_wrapper is not None:
        try:
            boundari_wrapper.save_audit()
        except Exception:
            pass  # Audit logging is best-effort

    return result.data
