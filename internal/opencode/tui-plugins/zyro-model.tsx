// @ts-nocheck
/** @jsxImportSource @opentui/solid */
/**
 * zyro-model — OpenCode TUI Plugin for per-agent model assignment.
 */

const tui = async (api) => {
  try {
    api.keymap.registerLayer({
      commands: [{ name: "zyro-model", title: "Zyro Model", run: () => { showAgentSelector(api); return true } }],
      bindings: [{ key: "alt+k", cmd: "zyro-model" }],
    })
  } catch (_) {}

  try {
    if (api.command) {
      api.command.register(() => [{
        title: "Zyro Model", value: "zyro-model", slash: { name: "zyro-model" },
        onSelect: () => showAgentSelector(api),
      }])
    }
  } catch (_) {}
}

// ---------------------------------------------------------------------------
// Agent catalog
// ---------------------------------------------------------------------------

const AGENTS = [
  { name: "zyro-orchestrator",     desc: "Coordinador — solo habla y delega",                 phase: "",     cat: "🌟 General" },
  { name: "to-issues",             desc: "Generación de GitHub Issues desde PRDs",            phase: "",     cat: "🌟 General" },
  { name: "zyro-pre-f0",           desc: "Alineación de dominio — grill-me, domain-model",     phase: "PRE",  cat: "🔍 PRE-F0: Alineación" },
  { name: "zyro-phase-0-patterns", desc: "Búsqueda de patrones similares",                    phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-phase-0-libraries",desc: "Investigación de librerías",                        phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-skills-find",      desc: "Descubrimiento de skills",                          phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-skills-audit",     desc: "Validación de skills descubiertas",                 phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-skills-apply",     desc: "Instalación de skills aprobadas",                   phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-sdd-explore",      desc: "Exploración de codebase y requerimientos",          phase: "F0",   cat: "📚 F0: Investigación" },
  { name: "zyro-sdd-spec",         desc: "Especificación técnica basada en hallazgos de F0",  phase: "F1",   cat: "📝 F1: Especificación" },
  { name: "zyro-sdd-propose",      desc: "Propuestas de cambio con alcance y enfoque",        phase: "F2",   cat: "🎨 F2: Diseño" },
  { name: "zyro-sdd-design",       desc: "Diseño técnico basado en Spec",                     phase: "F2",   cat: "🎨 F2: Diseño" },
  { name: "zyro-sdd-tasks",        desc: "División en tareas atómicas",                       phase: "F2",   cat: "🎨 F2: Diseño" },
  { name: "zyro-sdd-apply",        desc: "Implementación siguiendo specs, design y tasks",    phase: "F3",   cat: "⚡ F3: Implementación" },
  { name: "zyro-sdd-verify",       desc: "Verificación contra specs, design y tasks",         phase: "F3",   cat: "⚡ F3: Implementación" },
  { name: "zyro-sdd-archive",      desc: "Archivo de cambios completados",                    phase: "F4",   cat: "✅ F4: Cierre" },
]

// ---------------------------------------------------------------------------
// Dialog 1: Agent selector
// ---------------------------------------------------------------------------

function showAgentSelector(api) {
  api.ui.dialog.setSize("large")

  const providers = (api.state.provider || []).slice()
  // config is now read fresh inside AGENTS.map() below

  const options = [
    // 1. Set All (always first)
    {
      title: "★ Set All",
      value: "__SET_ALL__",
      description: "Asignar el mismo modelo a TODOS los agentes",
    },
    // 2. Agents with category grouping
    ...AGENTS.map((a) => {
      const config = api.state.config || {}  // read fresh inside map
      return {
        title: a.name,
        value: a.name,
        description: a.desc,
        footer: config?.agent?.[a.name]?.model ? `Modelo: ${config.agent[a.name].model}` : "Modelo: (hereda del orchestrator)",
        category: a.cat,
      }
    }),
    // 3. Done (always last)
    {
      title: "✓ Done — Terminar",
      value: "__DONE__",
      description: "Salir del configurador de modelos",
      category: "Acciones",
    },
  ]

  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: "ZyroCLI — Asignar Modelos",
      subtitle: "Paso 1/3 — Seleccioná un agente para configurar",
      placeholder: "Buscá un agente por nombre o fase...",
      options,
      flat: true,
      onSelect: (opt) => {
        if (opt.value === "__DONE__") { api.ui.dialog.clear(); return }
        showProviderSelector(api, providers, opt.value)
      },
    })
  )
}

// ---------------------------------------------------------------------------
// Dialog 2: Provider selector
// ---------------------------------------------------------------------------

function showProviderSelector(api, providers, agentName) {
  api.ui.dialog.setSize("large")

  if (!providers || providers.length === 0) {
    api.ui.toast({ message: "No hay proveedores. Usá /connect para agregar uno.", variant: "error" })
    showAgentSelector(api)
    return
  }

  const label = agentName === "__SET_ALL__" ? "★ Set All" : agentName

  const options = [
    ...providers.map((p) => {
      const models = p.models || {}
      const count = Array.isArray(models) ? models.length : Object.keys(models).length
      return {
        title: p.id || p.name,
        value: p.id || p.name,
        description: `${count} modelo${count !== 1 ? "s" : ""} disponible${count !== 1 ? "s" : ""}`,
        category: count > 0 ? "Disponibles" : "Sin modelos",
      }
    }),
    // Back at bottom
    {
      title: "← Volver a agentes",
      value: "__BACK__",
      description: "Volver al paso anterior",
      category: "Acciones",
    },
  ]

  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: `Proveedor para: ${label}`,
      subtitle: `Paso 2/3 — ${label} → ?`,
      placeholder: "Buscá un proveedor...",
      options,
      flat: true,
      onSelect: (opt) => {
        if (opt.value === "__BACK__") { showAgentSelector(api); return }
        showModelSelector(api, providers, agentName, opt.value)
      },
    })
  )
}

// ---------------------------------------------------------------------------
// Dialog 3: Model selector
// ---------------------------------------------------------------------------

function showModelSelector(api, providers, agentName, providerId) {
  api.ui.dialog.setSize("large")

  const provider = providers.find((p) => (p.id || p.name) === providerId)
  const models = provider?.models || {}
  const modelList = Array.isArray(models)
    ? models
    : Object.keys(models).map((id) => ({ id, name: models[id]?.name || id }))

  if (modelList.length === 0) {
    api.ui.toast({ message: `${providerId} no tiene modelos disponibles.`, variant: "error" })
    showProviderSelector(api, providers, agentName)
    return
  }

  const label = agentName === "__SET_ALL__" ? "★ Set All" : agentName

  const options = [
    ...modelList.map((m) => ({
      title: m.id || m.name,
      value: m.id || m.name,
    })),
    // Back at bottom
    {
      title: "← Volver a proveedores",
      value: "__BACK__",
      description: "Volver al paso anterior",
      category: "Acciones",
    },
  ]

  api.ui.dialog.replace(() =>
    api.ui.DialogSelect({
      title: `Modelo para: ${label}`,
      subtitle: `Paso 3/3 — ${label} → ${providerId}`,
      placeholder: "Buscá un modelo...",
      options,
      flat: true,
      onSelect: async (opt) => {
        if (opt.value === "__BACK__") { showProviderSelector(api, providers, agentName); return }
        await assignModel(api, providers, agentName, providerId, opt.value)
      },
    })
  )
}

// ---------------------------------------------------------------------------
// Agent model persistence via zyrocli
// ---------------------------------------------------------------------------

/**
 * Persists model assignments via zyrocli profile set.
 * Uses the full path to the zyrocli binary to bypass PATH issues
 * inside OpenCode's sandbox. Executes a real subprocess (Bun.$)
 * which writes to the real filesystem, not a virtual one.
 * Returns the path written on success, null on failure.
 * Never throws.
 */
async function persistAgentModel(agentName, modelStr) {
  try {
    // Get home directory via shell (reliable in any Bun runtime)
    const home = (await Bun.$`echo $HOME`.text()).trim()
    if (!home) return null
    
    // Find zyrocli binary — probe known locations
    let zyrocliPath = "zyrocli" // fallback
    const candidates = [
      `${home}/.local/bin/zyrocli`,
      `${home}/go/bin/zyrocli`,
      "/usr/local/bin/zyrocli",
      "/usr/bin/zyrocli",
    ]
    for (const p of candidates) {
      const found = await Bun.$`test -f ${p} && echo yes`.text()
      if (found.trim() === "yes") {
        zyrocliPath = p
        break
      }
    }
    
    if (agentName === "__SET_ALL__") {
      for (const a of AGENTS) {
        const result = await Bun.$`${zyrocliPath} profile set ${a.name} ${modelStr}`.text()
        console.log(`[zyro-model] profile set ${a.name} → ${result.trim()}`)
      }
    } else {
      const result = await Bun.$`${zyrocliPath} profile set ${agentName} ${modelStr}`.text()
      console.log(`[zyro-model] profile set ${agentName} → ${result.trim()}`)
    }
    
    // Also update AGENTS.md frontmatter if present (OpenCode reads model from there)
    try {
      const cwd = (await Bun.$`pwd`.text()).trim()
      if (cwd) {
        const agentsMd = `${cwd}/.config/opencode/AGENTS.md`
        const exists = await Bun.$`test -f ${agentsMd} && echo yes`.text()
        if (exists.trim() === "yes") {
          const escapedModel = modelStr.replace(/\|/g, '\\|')
          if (agentName === "__SET_ALL__") {
            for (const a of AGENTS) {
              await Bun.$`sed -i 's|^model:.*$|model: ${escapedModel}|' ${agentsMd}`.text()
            }
          } else {
            // For specific agent: only zyro-orchestrator has an AGENTS.md
            if (agentName === "zyro-orchestrator") {
              await Bun.$`sed -i 's|^model:.*$|model: ${escapedModel}|' ${agentsMd}`.text()
            }
          }
        }
      }
    } catch (_) {
      // Non-fatal: AGENTS.md update is best-effort
    }
    
    // Return the config path that zyrocli wrote to (use GetEffectiveConfigPath logic)
    // Try project config first, then global
    try {
      const cwd = (await Bun.$`pwd`.text()).trim()
      if (cwd) {
        const projectConfig = `${cwd}/.config/opencode/opencode.json`
        const testResult = await Bun.$`test -f ${projectConfig} && echo exists`.text()
        if (testResult.trim() === "exists") return projectConfig
      }
    } catch (_) {}
    return `${home}/.config/opencode/opencode.json`
  } catch (err) {
    console.error("[zyro-model] persistAgentModel failed:", err)
    return null
  }
}

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

async function assignModel(api, providers, agentName, providerId, modelId) {
  const modelStr = `${providerId}/${modelId}`

  try {
    const updates = {}
    if (agentName === "__SET_ALL__") {
      for (const a of AGENTS) updates[a.name] = { model: modelStr }
    } else {
      updates[agentName] = { model: modelStr }
    }
    await api.client.global.config.update({ body: { agent: updates } })

    // Force-refresh api.state.config so the next showAgentSelector reads fresh data
    if (!api.state.config) api.state.config = {}
    if (!api.state.config.agent) api.state.config.agent = {}
    if (agentName === "__SET_ALL__") {
      for (const a of AGENTS) {
        api.state.config.agent[a.name] = api.state.config.agent[a.name] || {}
        api.state.config.agent[a.name].model = modelStr
      }
    } else {
      api.state.config.agent[agentName] = api.state.config.agent[agentName] || {}
      api.state.config.agent[agentName].model = modelStr
    }

    // 3. Persistir via zyrocli profile set (real subprocess, real filesystem)
    const savedPath = await persistAgentModel(agentName, modelStr)

    const label = agentName === "__SET_ALL__" ? "Todos los agentes" : agentName
    if (savedPath) {
      api.ui.toast({
        message: `✓ ${label} → ${modelStr} (📁 ${savedPath})`,
        variant: "success",
      })
    } else {
      api.ui.toast({
        message: `✓ ${label} → ${modelStr} ⚠️ no se pudo persistir a archivo`,
        variant: "warning",
      })
    }
  } catch (err) {
    api.ui.toast({ message: `✗ Error: ${err?.message || "desconocido"}`, variant: "error" })
  }

  showAgentSelector(api)
}

const plugin = { id: "zyro-model", tui }
export default plugin
