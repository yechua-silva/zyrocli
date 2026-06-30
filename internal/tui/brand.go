package tui

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colores de la marca Zyro
	colorNaranja = lipgloss.Color("#F97316")
	colorVerde   = lipgloss.Color("#10B981")
	colorGris    = lipgloss.Color("#52525B")
	colorBlanco  = lipgloss.Color("#FFFFFF")

	// brandStyle para el logo ZYRO 3D
	brandStyle = lipgloss.NewStyle().
			Foreground(colorNaranja).
			Bold(true)

	// logoStyle para el Zorro Hacker (usado solo en OpenCode, no en CLI)
	logoStyle = lipgloss.NewStyle().
			Foreground(colorVerde)

	// logoNewStyle para el nuevo zorro naranja
	logoNewStyle = lipgloss.NewStyle().
			Foreground(colorNaranja).
			Bold(true)

	// subtitleStyle para texto secundario
	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorGris).
			Italic(true)

	// stepGroupStyle para agrupar pasos de instalación (con borde redondeado)
	stepGroupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorNaranja).
			Padding(1, 3)

	// successStyle para mensajes de éxito
	successStyle = lipgloss.NewStyle().
			Foreground(colorVerde).
			Bold(true)

	// warningStyle para advertencias
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	// errorStyle para errores
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	// infoStyle para información
	infoStyle = lipgloss.NewStyle().
			Foreground(colorNaranja)

	// labelStyle para etiquetas (ej: "✓ MCP tools:")
	labelStyle = lipgloss.NewStyle().
			Foreground(colorBlanco).
			Bold(true)

	// centeredStyle para centrar contenido horizontalmente
	centeredStyle = lipgloss.NewStyle().
			Align(lipgloss.Center)
)

//go:embed assets/brand.txt
var brandArtRaw string

//go:embed assets/logo.txt
var logoArtRaw string

//go:embed assets/logo-new.txt
var logoNewArtRaw string

// brandArt es la versión sanitizada (sin trailing spaces) del arte ZYRO 3D.
var brandArt string

// logoArt es la versión sanitizada del Zorro Hacker.
var logoArt string

// logoNewArt es la versión sanitizada del nuevo zorro naranja.
var logoNewArt string

func init() {
	brandArt = sanitizeArt(brandArtRaw)
	logoArt = sanitizeArt(logoArtRaw)
	logoNewArt = sanitizeArt(logoNewArtRaw)
}

func sanitizeArt(art string) string {
	return strings.ReplaceAll(art, "\r\n", "\n")
}

// maxLineWidth calcula el ancho máximo visual de un arte ASCII (sin trailing spaces).
func maxLineWidth(art string) int {
	max := 0
	for _, line := range strings.Split(art, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if len(trimmed) > max {
			max = len(trimmed)
		}
	}
	return max
}

// centeredBlock centra un bloque de texto ASCII horizontalmente.
// NOTA: No usa bordes. El arte respira por sí solo.
func centeredBlock(art string, style lipgloss.Style) string {
	art = sanitizeArt(art)
	lines := strings.Split(strings.TrimSuffix(art, "\n"), "\n")
	artWidth := maxLineWidth(art)
	styledLines := make([]string, len(lines))
	for i, line := range lines {
		// Padding derecho para que todas las líneas tengan el mismo ancho
		line = strings.TrimRight(line, " ")
		padded := line + strings.Repeat(" ", artWidth-len(line))
		styledLines[i] = style.Render(padded)
	}
	return strings.Join(styledLines, "\n")
}

// ── Banner principal ────────────────────────────────────────────────────

// RenderBrand renderiza el logo ZYRO 3D centrado con estilo naranja (sin borde).
func RenderBrand() string {
	return centeredBlock(brandArt, brandStyle)
}

// RenderLogo renderiza el Zorro Hacker centrado con estilo verde.
// Solo se usa para el tema de OpenCode, NO en el CLI install.
func RenderLogo() string {
	return centeredBlock(logoArt, logoStyle)
}

// RenderNewLogo renderiza el nuevo zorro naranja.
func RenderNewLogo() string {
	return centeredBlock(logoNewArt, logoNewStyle)
}

// RenderWelcome renderiza brand + subtítulo centrados, SIN borde.
// Es el banner principal para el flujo de instalación.
func RenderWelcome(subtitle string) string {
	return lipgloss.JoinVertical(lipgloss.Center,
		RenderBrand(),
		"",
		subtitleStyle.Render(subtitle),
	)
}

// RenderFullBanner es idéntico a RenderWelcome (sin Zorro, sin borde).
// Se mantiene para compatibilidad pero se recomienda usar RenderWelcome.
func RenderFullBanner(subtitle string) string {
	return RenderWelcome(subtitle)
}

// ── Small banner (for narrow terminals) ─────────────────────────────────

// smallBrandArt is a minimal text-only brand for terminals <60 columns.
const smallBrandArt = `ZyroCLI`

// RenderSmallBanner renders a minimal text-only banner for narrow terminals (sin borde).
func RenderSmallBanner(subtitle string) string {
	smallStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorNaranja).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(lipgloss.Center,
		smallStyle.Render(smallBrandArt),
		"",
		subtitleStyle.Render(subtitle),
	)
}

// ── Separador visual ─────────────────────────────────────────────────────

// RenderSeparator dibuja una línea horizontal sutil (ej: ─────────────).
func RenderSeparator() string {
	line := strings.Repeat("─", 40)
	return lipgloss.NewStyle().
		Foreground(colorGris).
		Render(line)
}

// ── Mensajes de estado ──────────────────────────────────────────────────

// Success renderiza un mensaje de éxito con checkmark verde.
func Success(text string) string {
	return fmt.Sprintf("  %s %s", successStyle.Render("✓"), text)
}

// Warning renderiza un mensaje de advertencia.
func Warning(text string) string {
	return fmt.Sprintf("  %s %s", warningStyle.Render("⚠"), text)
}

// ErrorStr renderiza un mensaje de error.
func ErrorStr(text string) string {
	return fmt.Sprintf("  %s %s", errorStyle.Render("✗"), text)
}

// Info renderiza un mensaje informativo.
func Info(text string) string {
	return fmt.Sprintf("  %s %s", infoStyle.Render("•"), text)
}

// PrintSuccess imprime un mensaje de éxito.
func PrintSuccess(text string) {
	fmt.Println(Success(text))
}

// PrintWarning imprime una advertencia.
func PrintWarning(text string) {
	fmt.Println(Warning(text))
}

// PrintError imprime un error.
func PrintError(text string) {
	fmt.Println(ErrorStr(text))
}

// PrintInfo imprime un mensaje informativo.
func PrintInfo(text string) {
	fmt.Println(Info(text))
}

// PrintBrand imprime brand centrado sin borde en stdout.
func PrintBrand(subtitle string) {
	fmt.Println(RenderWelcome(subtitle))
}

// PrintFullBanner imprime brand sin Zorro ni borde en stdout (compatibilidad).
func PrintFullBanner(subtitle string) {
	fmt.Println(RenderFullBanner(subtitle))
}

// BrandLines retorna las líneas del brand (para bubbletea).
func BrandLines() []string {
	return strings.Split(brandArt, "\n")
}

// LogoLines retorna las líneas del logo (para bubbletea).
func LogoLines() []string {
	return strings.Split(strings.TrimSuffix(logoArt, "\n"), "\n")
}
