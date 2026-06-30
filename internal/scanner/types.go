package scanner

import "time"

// Language representa un lenguaje de programación.
type Language string

const (
	LanguageGo         Language = "Go"
	LanguageTypeScript Language = "TypeScript"
	LanguageJavaScript Language = "JavaScript"
	LanguageRust       Language = "Rust"
	LanguagePython     Language = "Python"
	LanguageRuby       Language = "Ruby"
	LanguagePHP        Language = "PHP"
	LanguageC          Language = "C"
	LanguageCpp        Language = "C++"
	LanguageCSharp     Language = "C#"
	LanguageJava       Language = "Java"
	LanguageElixir     Language = "Elixir"
	LanguageSwift      Language = "Swift"
	LanguageKotlin     Language = "Kotlin"
	LanguageUnknown    Language = "Unknown"
)

// FileInfo contiene metadatos de un archivo escaneado.
type FileInfo struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Ext      string   `json:"ext"`
	Size     int64    `json:"size"`
	Hash     string   `json:"hash"`
	Language Language `json:"language"`
}

// ProjectInfo contiene el resultado completo del escaneo.
type ProjectInfo struct {
	Root      string     `json:"root"`
	Name      string     `json:"name"`
	Language  Language   `json:"language"`
	Files     []FileInfo `json:"files"`
	FileCount int        `json:"file_count"`
	TotalSize int64      `json:"total_size"`
	ScannedAt time.Time  `json:"scanned_at"`
}
