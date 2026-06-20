// @ts-nocheck
/** @jsxImportSource @opentui/solid */
/**
 * zyro-model — OpenCode TUI Plugin for per-agent model assignment.
 *
 * API: @opencode-ai/plugin/tui v1
 * Location: ~/.config/opencode/tui-plugins/zyro-model.tsx
 */

const tui = async (api) => {
  // --- Register via keymap (Alt+K) ---
  try {
    api.keymap.registerLayer({
      commands: [{
        name: "zyro-model",
        title: "Zyro Model",
        run: () => {
          showAgentSelector(api)
          return true
        },
      }],
      bindings: [{ key: "alt+k", cmd: "zyro-model" }],
    })
  } catch (err) {
    console.error("[zyro-model] keymap error:", err)
  }

  // --- Register via legacy command (/zyro-model) ---
  try {
    if (api.command) {
      api.command.register(() => [{
        title: "Zyro Model",
        value: "zyro-model",
        slash: { name: "zyro-model" },
        onSelect: () => showAgentSelector(api),
      }])
    }
  } catch (err) {
    console.error("[zyro-model] command error:", err)
  }
}

// ---------------------------------------------------------------------------
// Agent list with Set All + Done
// ---------------------------------------------------------------------------

const AGENTS = [
  { name: "zyro-orchestrator",     desc: "Coordinador — solo habla y delega",                 phase: "" },
  { name: "zyro-pre-f0",           desc: "PRE-F0: Alineación de dominio",                     phase: "PRE-F0" },
  { name: "zyro-phase-0-patterns", desc: "F0: Búsqueda de patrones similares",                phase: "F0" },
  { name: "zyro-phase-0-libraries",desc: "F0: Investigación de librerías",                    phase: "F0" },
  { name: "zyro-skills-find",      desc: "F0: Descubrimiento de skills",                      phase: "F0" },
  { name: "zyro-skills-audit",     desc: "F0: Validación de skills descubiertas",             phase: "F0" },
  { name: "zyro-skills-apply",     desc: "F0: Instalación de skills aprobadas",               phase: "F0" },
  { name: "zyro-sdd-explore",      desc: "F0: Exploración de codebase y requerimientos",      phase: "F0" },
  { name: "zyro-sdd-spec",         desc: "F1: Especificación técnica",                        phase: "F1" },
  { name: "zyro-sdd-propose",      desc: "F2: Propuestas de cambio",                          phase: "F2" },
  { name: "zyro-sdd-design",       desc: "F2: Diseño técnico basado en Spec",                 phase: "F2" },
  { name: "zyro-sdd-tasks",        desc: "F2: División en tareas atómicas",                   phase: "F2" },
  { name: "zyro-sdd-apply",        desc: "F3: Implementación siguiendo specs, design y tasks", phase: "F3" },
  { name: "zyro-sdd-verify",       desc: "F3: Verificación contra specs y design",            phase: "F3" },
  { name: "zyro-sdd-archive",      desc: "F4: Archivo de cambios completados",                phase: "F4" },
  { name: "to-issues",             desc: "Generación de GitHub Issues desde PRDs",            phase: "" },
]

const PHASE_ORDER = { "": 0, "PRE-F0": 1, "F0": 2, "F1": 3, "F2": 4, "F3": 5, "F4": 6 }

function showAgentSelector(api) {
  const providers = (api.state.provider || []).slice()
  const config = api.state.config || {}

  // Sort agents
  const sorted = [...AGENTS].sort((a, b) => {
    const pa = PHASE_ORDER[a.phase] ?? 99
    const pb = PHASE_ORDER[b.phase] ?? 99
    if (pa !== pb) return pa - pb
    return a.name.localeCompare(b.name)
  })

  // Build options
  const options = []

  // Set All
  options.push({ title: "★ Set All", value: "__SET_ALL__", description: "Asignar el mismo modelo a TODOS los agentes" })

  // Agents
  for (const a of sorted) {
    const phase = a.phase ? `[${a.phase}]` : ""
    const model = config?.agent?.[a.name]?.model ? `Actual: ${config.agent[a.name].model}` : "hereda del orchestrator"
    options.push({
      title: phase ? `${phase} ${a.name}` : a.name,
      value: a.name,
      description: `${a.desc} — ${model}`,
    })
  }

  // Done
  options.push({ title: "✓ Done — Terminar", value: "__DONE__", description: "Salir del configurador" })

  // Show DialogSelect via dialog.replace()
  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: "ZyroCLI — Selector de Modelos",
      options,
      flat: true,
      onSelect: (opt) => {
        if (opt.value === "__DONE__") {
          api.ui.dialog.clear()
          return
        }
        // Show providers next
        showProviderSelector(api, providers, config, opt.value)
      },
    })
  )
}

function showProviderSelector(api, providers, config, agentName) {
  if (!providers || providers.length === 0) {
    api.ui.toast({ message: "No hay proveedores. Usá /connect para agregar uno.", variant: "error" })
    showAgentSelector(api)
    return
  }

  const options = [
    { title: "← Volver a agentes", value: "__BACK__" },
    ...providers.map((p) => ({
      title: p.id || p.name,
      value: p.id || p.name,
      description: `${(p.models ? Object.keys(p.models).length : 0) || 0} modelos`,
    })),
  ]

  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: `Proveedor para: ${agentName}`,
      options,
      flat: true,
      onSelect: (opt) => {
        if (opt.value === "__BACK__") {
          showAgentSelector(api)
          return
        }
        showModelSelector(api, providers, config, agentName, opt.value)
      },
    })
  )
}

function showModelSelector(api, providers, config, agentName, providerId) {
  const provider = providers.find((p) => (p.id || p.name) === providerId)
  const models = provider?.models || {}
  const modelList = Array.isArray(models) ? models : Object.keys(models).map((id) => ({ id, name: models[id]?.name || id }))

  if (modelList.length === 0) {
    api.ui.toast({ message: `${providerId} no tiene modelos.`, variant: "error" })
    showProviderSelector(api, providers, config, agentName)
    return
  }

  const options = [
    { title: "← Volver a proveedores", value: "__BACK__" },
    ...modelList.map((m) => ({ title: m.id || m.name, value: m.id || m.name })),
  ]

  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: `Modelo de ${providerId} para: ${agentName}`,
      options,
      flat: true,
      onSelect: async (opt) => {
        if (opt.value === "__BACK__") {
          showProviderSelector(api, providers, config, agentName)
          return
        }
        await assignModel(api, providers, config, agentName, providerId, opt.value)
      },
    })
  )
}

async function assignModel(api, providers, config, agentName, providerId, modelId) {
  const modelStr = `${providerId}/${modelId}`

  try {
    // Runtime
    const updates = {}
    if (agentName === "__SET_ALL__") {
      for (const a of AGENTS) updates[a.name] = { model: modelStr }
    } else {
      updates[agentName] = { model: modelStr }
    }
    await api.client.global.config.update({ body: { agent: updates } })

    // Persist
    if (agentName === "__SET_ALL__") {
      for (const a of AGENTS) {
        await Bun.$`zyrocli profile set ${a.name} ${modelStr}`.text()
      }
    } else {
      await Bun.$`zyrocli profile set ${agentName} ${modelStr}`.text()
    }

    api.ui.toast({
      message: `✓ ${agentName === "__SET_ALL__" ? "Todos" : agentName} → ${modelStr}`,
      variant: "success",
    })
  } catch (err) {
    api.ui.toast({ message: `✗ Error: ${err?.message || "desconocido"}`, variant: "error" })
  }

  // Loop back
  showAgentSelector(api)
}

const plugin = { id: "zyro-model", tui }
export default plugin
