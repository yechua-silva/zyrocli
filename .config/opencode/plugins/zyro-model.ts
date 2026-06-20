/**
 * zyro-model — OpenCode slash command for per-agent model assignment.
 *
 * Usage:  /zyro-model
 *
 * Shows current model assignments per ZyroCLI agent and provides
 * instructions to configure models via the zyrocli CLI.
 *
 * The actual interactive TUI runs via `zyrocli profile tui` (bubbletea);
 * this plugin provides the in-OpenCode convenience entry point.
 *
 * Plugin API used:
 *   - client.config.providers()   — list available providers/models
 *   - client.config.get()         — read current agent config
 *   - client.tui.showToast()      — show confirmation
 *   - client.tui.appendPrompt()   — pre-fill prompt for convenience
 */

import type { Plugin } from "@opencode-ai/plugin";

// ---------------------------------------------------------------------------
// ZyroCLI agent definitions (mirrors profile_agents.go)
// ---------------------------------------------------------------------------

interface AgentInfo {
  name: string;
  description: string;
  phase: string;
}

const ZYRO_AGENTS: AgentInfo[] = [
  { name: "zyro-orchestrator",     description: "Coordinador — solo habla y delega",                 phase: "" },
  { name: "zyro-pre-f0",           description: "PRE-F0: Alineación de dominio",                     phase: "PRE-F0" },
  { name: "zyro-phase-0-patterns", description: "F0: Búsqueda de patrones similares",                phase: "F0" },
  { name: "zyro-phase-0-libraries",description: "F0: Investigación de librerías",                    phase: "F0" },
  { name: "zyro-skills-find",      description: "F0: Descubrimiento de skills",                      phase: "F0" },
  { name: "zyro-skills-audit",     description: "F0: Validación de skills descubiertas",             phase: "F0" },
  { name: "zyro-skills-apply",     description: "F0: Instalación de skills aprobadas",               phase: "F0" },
  { name: "zyro-sdd-explore",      description: "F0: Exploración de codebase y requerimientos",      phase: "F0" },
  { name: "zyro-sdd-spec",         description: "F1: Especificación técnica",                        phase: "F1" },
  { name: "zyro-sdd-propose",      description: "F2: Propuestas de cambio",                          phase: "F2" },
  { name: "zyro-sdd-design",       description: "F2: Diseño técnico basado en Spec",                 phase: "F2" },
  { name: "zyro-sdd-tasks",        description: "F2: División en tareas atómicas",                   phase: "F2" },
  { name: "zyro-sdd-apply",        description: "F3: Implementación siguiendo specs, design y tasks", phase: "F3" },
  { name: "zyro-sdd-verify",       description: "F3: Verificación contra specs y design",            phase: "F3" },
  { name: "zyro-sdd-archive",      description: "F4: Archivo de cambios completados",                phase: "F4" },
  { name: "to-issues",             description: "Generación de GitHub Issues desde PRDs",            phase: "" },
];

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

export const ZyroModelPlugin: Plugin = async ({ client }) => {
  return {
    command: {
      /**
       * /zyro-model — Show model assignments and provide instructions.
       * Supports:
       *   /zyro-model list              — show current assignments
       *   /zyro-model set <agent> <m>   — set model (via zyrocli)
       *   /zyro-model set-all <model>   — set same model for all
       */
      "zyro-model": async (args: string) => {
        const parts = (args || "").trim().split(/\s+/);
        const sub = parts[0]?.toLowerCase();

        if (sub === "list" || !sub) {
          await showAgentList(client);
        } else if (sub === "set" && parts.length >= 3) {
          const agentName = parts[1];
          const modelStr = parts.slice(2).join(" ");
          await setAgentModel(client, agentName, modelStr);
        } else if (sub === "set-all" && parts.length >= 2) {
          const modelStr = parts.slice(1).join(" ");
          await setAllAgentsModel(client, modelStr);
        } else {
          await showHelp(client);
        }
      },
    },
  };
};

// ---------------------------------------------------------------------------
// Command handlers
// ---------------------------------------------------------------------------

async function showAgentList(client: any) {
  const { providers } = await client.config.providers();
  const config = await client.config.get();
  const agentConfig = config?.agent || {};

  let text = `## 🤖 ZyroCLI — Model Assignments\n\n`;
  text += `| Agent | Fase | Modelo Actual |\n`;
  text += `|-------|------|--------------|\n`;

  for (const agent of ZYRO_AGENTS) {
    const current = agentConfig[agent.name]?.model || "*(hereda del orchestrator)*";
    const phaseTag = agent.phase || "—";
    text += `| \`${agent.name}\` | ${phaseTag} | ${current} |\n`;
  }

  text += `\n### How to set models\n\n`;
  text += "```\n";
  text += "# For a single agent:\n";
  text += "zyrocli profile set <agent> <provider/model>\n\n";
  text += "# Interactive TUI (recommended for first-time setup):\n";
  text += "zyrocli profile tui\n\n";
  text += "# Examples:\n";
  text += `zyrocli profile set zyro-sdd-apply anthropic/claude-sonnet-4\n`;
  text += `zyrocli profile set zyro-sdd-verify google/gemini-2.5-pro\n`;
  text += "```\n\n";

  if (providers && providers.length > 0) {
    text += `### Available providers\n\n`;
    for (const p of providers) {
      const modelList = (p.models || []).map((m: any) => `\`${m.id}\``).join(", ");
      text += `- **${p.id}** (${p.name}): ${modelList}\n`;
    }
  }

  text += `\n---\n`;
  text += `💡 Tip: También podés usar \`zyrocli profile list\` desde cualquier terminal.\n`;

  // Append to the conversation
  await client.session.prompt({
    body: {
      noReply: true,
      parts: [{ type: "text", text }],
    },
  });

  await client.tui.showToast({
    message: "✓ /zyro-model: lista de asignaciones generada",
    variant: "success",
  });
}

async function setAgentModel(client: any, agentName: string, modelStr: string) {
  const config = await client.config.get();
  const agentConfig = config?.agent || {};

  if (!ZYRO_AGENTS.some((a) => a.name === agentName)) {
    await showError(client, `Agente "${agentName}" no encontrado. Usá /zyro-model list para ver los agentes.`);
    return;
  }

  // Validate format provider/model
  if (!modelStr.includes("/")) {
    await showError(client, `Formato inválido: "${modelStr}". Debe ser provider/model (ej: anthropic/claude-sonnet-4)`);
    return;
  }

  // Append a command to the prompt so the user can run it
  await client.tui.appendPrompt({
    body: {
      text: `!zyrocli profile set ${agentName} ${modelStr}`,
    },
  });

  await client.tui.showToast({
    message: `Pre-cargado: zyrocli profile set ${agentName} ${modelStr} — presioná Enter para ejecutar`,
    variant: "info",
  });
}

async function setAllAgentsModel(client: any, modelStr: string) {
  if (!modelStr.includes("/")) {
    await showError(client, `Formato inválido: "${modelStr}". Debe ser provider/model (ej: anthropic/claude-sonnet-4)`);
    return;
  }

  // Build a multi-set command
  let cmd = "";
  for (const agent of ZYRO_AGENTS) {
    cmd += `zyrocli profile set ${agent.name} ${modelStr} && `;
  }
  cmd = cmd.replace(/ && $/, "");

  await client.tui.appendPrompt({
    body: {
      text: `!${cmd}`,
    },
  });

  await client.tui.showToast({
    message: `Pre-cargado: Set All → ${modelStr} — presioná Enter para ejecutar`,
    variant: "info",
  });
}

async function showHelp(client: any) {
  const text = `## /zyro-model — Ayuda\n\n`;
  const helpText = text + [
    "Usage:",
    "  /zyro-model                  — Mostrar asignaciones actuales",
    "  /zyro-model list             — Mostrar asignaciones actuales",
    `  /zyro-model set <agent> <m>  — Pre-cargar comando para asignar modelo`,
    `  /zyro-model set-all <model>  — Pre-cargar comando para mismo modelo en todos`,
    "",
    "Ejemplos:",
    `  /zyro-model set zyro-sdd-apply anthropic/claude-sonnet-4`,
    `  /zyro-model set-all opencode-go/deepseek-v4-flash`,
    "",
    "Agentes disponibles:",
    ...ZYRO_AGENTS.map((a) => `  ${a.name} — ${a.description}`),
  ].join("\n");

  await client.session.prompt({
    body: {
      noReply: true,
      parts: [{ type: "text", text: helpText }],
    },
  });
}

async function showError(client: any, message: string) {
  await client.session.prompt({
    body: {
      noReply: true,
      parts: [{ type: "text", text: `❌ ${message}` }],
    },
  });
  await client.tui.showToast({
    message: `❌ ${message}`,
    variant: "error",
  });
}

export default ZyroModelPlugin;
