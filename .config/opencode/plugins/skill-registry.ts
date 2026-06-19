/**
 * skill-registry — no-op
 *
 * Previously refreshed Gentle AI's project skill registry at startup.
 * Gentle AI is no longer used; this plugin now does nothing.
 */

import type { Plugin } from "@opencode-ai/plugin"

const SkillRegistryPlugin: Plugin = async () => {
  return {}
}

export default SkillRegistryPlugin
