package main

// AgentDef defines a ZyroCLI agent entry for model assignment.
type AgentDef struct {
	Name        string // Agent ID in opencode.json
	Description string // Brief description shown in TUI
	Phase       string // SDD phase (PRE-F0, F0, F1, F2, F3, F4, or "")
	DefaultMode string // "primary" | "subagent"
}

// zyroAgents is the canonical list of all configurable ZyroCLI agents.
var zyroAgents = []AgentDef{
	{Name: "zyro-orchestrator",    Description: "Coordinador — solo habla y delega",                Phase: "",       DefaultMode: "primary"},
	{Name: "zyro-pre-f0",          Description: "PRE-F0: Alineación de dominio",                     Phase: "PRE-F0", DefaultMode: "subagent"},
	{Name: "zyro-phase-0-patterns",Description: "F0: Búsqueda de patrones similares",                Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-phase-0-libraries",Description: "F0: Investigación de librerías",                   Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-skills-find",     Description: "F0: Descubrimiento de skills",                      Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-skills-audit",    Description: "F0: Validación de skills descubiertas",             Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-skills-apply",    Description: "F0: Instalación de skills aprobadas",               Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-explore",     Description: "F0: Exploración de codebase y requerimientos",      Phase: "F0",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-spec",        Description: "F1: Especificación técnica",                        Phase: "F1",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-propose",     Description: "F2: Propuestas de cambio",                          Phase: "F2",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-design",      Description: "F2: Diseño técnico basado en Spec",                 Phase: "F2",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-tasks",       Description: "F2: División en tareas atómicas",                   Phase: "F2",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-apply",       Description: "F3: Implementación siguiendo specs, design y tasks", Phase: "F3",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-verify",      Description: "F3: Verificación contra specs y design",            Phase: "F3",     DefaultMode: "subagent"},
	{Name: "zyro-sdd-archive",     Description: "F4: Archivo de cambios completados",                Phase: "F4",     DefaultMode: "subagent"},
	{Name: "to-issues",            Description: "Generación de GitHub Issues desde PRDs",             Phase: "",       DefaultMode: "subagent"},
}
