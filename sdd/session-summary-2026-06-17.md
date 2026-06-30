# Session Summary — 2026-06-17

## Goal
Fix 3 issues in ZyroCLI: GPU installer automation, HelixDB post-install, subagent delegation. Plus research OpenCode async capabilities.

## User
- Name: Yechua
- Agent: Zyro (the AI)

## Instructions
- Do NOT touch the TUI visual — leave as bug, human will fix
- Do NOT use Context7 — Context + GitMCP is the stack
- Engram binary and MCP must be removed from system (the Go code in internal/memory/ is ours, keep it)
- Everything must be embedded in the binary (not manual config changes)
- Use the Zyro ecosystem (task board, skills, HelixDB)
- Respond in same language as human

## Discoveries
- OpenCode v1.17.8 has a full HTTP API when started with `opencode serve --port X`
- Endpoints: /session, /experimental/session/{id}/background, /tui/show-toast, /tui/append-prompt, /tui/submit-prompt
- SSE at /event and /global/event
- Full OpenAPI 3.1 spec at GET /doc
- Subagents cannot be invoked via CLI (no `opencode subagent` command)
- MCP_DIR_PLACEHOLDER must be resolved by install code, not manual edits

## Accomplished
- [x] Removed engram binary from ~/.local/bin/
- [x] Removed context7 and engram MCP from ~/.claude/mcp/
- [x] Removed engram plugin from ~/.claude/settings.json and .codex/config.toml
- [x] Removed ~/.claude/CLAUDE.md
- [x] Fixed MCP_DIR_PLACEHOLDER in install.go source + existing opencode.jsonc
- [x] Fixed GPU installer: _auto_configure_rocm(), _auto_configure_vulkan() in scripts/install_tui.py
- [x] Fixed HelixDB post-install: startHelixContainer(), EnsureStarted() improved, install step now fatal
- [x] Fixed subagents: executeTask no longer uses non-existent CLI, added CompleteTask(), DelegateStep fixed
- [x] Added complete_task tool to mcp-server
- [x] Created zyrocli watcher command with HelixDB polling + OpenCode API integration
- [x] Updated 14 embedded SKILL.md files with Notification step
- [x] dispatch_task now saves Task node to HelixDB
- [x] Updated README with watcher, notifications, HTTP API docs
- [x] Binary built (17MB Go) and installed in PATH

## Next Steps
- Make watcher auto-start with `zyrocli install` (transparent)
- Test the GPU installer end-to-end on real hardware
- Complete the Notification → TUI injection pipeline

## Relevant Files
- scripts/install_tui.py — GPU auto-configuration (lines 664-771 new functions)
- internal/db/helix/client.go — startHelixContainer, EnsureStarted improved
- internal/boomerang/task_manager.go — executeTask fixed, CompleteTask added
- internal/boomerang/delegate.go — no CLI subagent execution
- cmd/zyrocli/mcp_server.go — complete_task tool, dispatch_task saves to HelixDB
- cmd/zyrocli/watcher.go — new watcher command
- internal/watcher/watcher.go — watcher logic with HelixDB + OpenCode API
- internal/opencode/skills/*/SKILL.md — Notification step added
- cmd/zyrocli/install.go — MCP_DIR fixed, HelixDB fatal step
- sdd/spec-fix-3issues.md — Spec
- sdd/design-fix-3issues.md — Design + Tasks
- sdd/archive-fix-3issues.md — Archive record
- README.md — updated with watcher and notifications
