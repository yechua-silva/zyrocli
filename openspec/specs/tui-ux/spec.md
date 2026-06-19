# TUI UX Specification — Delta

## ADDED Requirements

### Requirement: New Orange Logo (ASCII art fox)

The TUI MUST embed a new ASCII art logo of an orange fox ("zorro naranja") via `//go:embed` in `internal/tui/brand.go`, and expose a `RenderNewLogo()` function that returns the styled centered art.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Logo rendered | `RenderNewLogo()` called | Function returns | Non-empty string with ASCII characters |
| Style applied | Returned string inspected | Color style is applied | Uses `colorNaranja` with Bold |
| Asset embedded | Binary compiled | Logo asset checked | Compiles without file system dependency |

### Requirement: Clear Screen Before Brand Display

The TUI MUST clear the terminal screen (`\033[2J\033[H`) before displaying the brand logo in `runSetupFlow()` and `runAutostartFlow()` to prevent duplicate/corrupt logo rendering.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fix bug #1 — setup flow | `runSetupFlow()` called | Terminal cleared before `RenderBrand()` | Screen is clean before brand |
| Fix bug #1 — autostart flow | `runAutostartFlow()` called | Terminal cleared before `RenderBrand()` | Screen is clean before brand |

### Requirement: Systematic Screen Clearing for Navigation

The TUI MUST clear the terminal screen at each navigation boundary: before the menu loop, before each flow (install, setup, models, autostart), and before confirm prompts within flows.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Menu loop starts | `handleMenu()` called | Screen cleared | Clean state before each menu iteration |
| Install flow starts | User selects install | Screen cleared | Before `installCmd.RunE()` |
| Install confirm | Install completes, confirm prompt | Screen cleared | Before `RunConfirm()` |
| Setup flow starts | User selects setup | Screen cleared | Before checks |
| Models flow starts | User selects models | Screen cleared | Before Ollama check |
| Models confirm test | Models loaded, confirm to test | Screen cleared | Before `RunConfirm()` |
| Models test | Test confirmed, tests run | Screen cleared | Before `TestEmbedding()` |
| Autostart flow starts | User selects autostart | Screen cleared | Before autostart setup |

### Requirement: About ZyroCLI Menu

The main menu MUST include a "📖 Acerca de ZyroCLI" (About) option that displays the brand logo followed by descriptive text about ZyroCLI.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Menu item visible | Menu renders | About option listed | "📖 Acerca de ZyroCLI" appears in options |
| About selected | User picks About | Flow executes | Brand + descriptive text displayed |
| About flow exits | User finishes reading | Flow returns | Back to main menu |
