package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var DefaultIgnorePatterns = []string{
	".git", ".hg", ".svn",
	"node_modules", "vendor", "dist", "build", "target",
	".venv", "__pycache__",
	".idea", ".vscode",
	".zyro", ".helix",
}

var extLangMap = map[string]Language{
	".go":    LanguageGo,
	".ts":    LanguageTypeScript,
	".tsx":   LanguageTypeScript,
	".js":    LanguageJavaScript,
	".jsx":   LanguageJavaScript,
	".mjs":   LanguageJavaScript,
	".cjs":   LanguageJavaScript,
	".rs":    LanguageRust,
	".py":    LanguagePython,
	".rb":    LanguageRuby,
	".php":   LanguagePHP,
	".c":     LanguageC,
	".h":     LanguageC,
	".cpp":   LanguageCpp,
	".cc":    LanguageCpp,
	".cxx":   LanguageCpp,
	".hpp":   LanguageCpp,
	".cs":    LanguageCSharp,
	".java":  LanguageJava,
	".kt":    LanguageKotlin,
	".kts":   LanguageKotlin,
	".ex":    LanguageElixir,
	".exs":   LanguageElixir,
	".swift": LanguageSwift,
}

type ProjectScanner struct {
	ignorePatterns []string
	maxFiles       int
}

func NewScanner() *ProjectScanner {
	return &ProjectScanner{
		ignorePatterns: DefaultIgnorePatterns,
		maxFiles:       10000,
	}
}

func (s *ProjectScanner) WithIgnorePatterns(p []string) *ProjectScanner {
	s.ignorePatterns = p
	return s
}

func (s *ProjectScanner) WithMaxFiles(n int) *ProjectScanner {
	s.maxFiles = n
	return s
}

func (s *ProjectScanner) Scan(root string) (*ProjectInfo, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Verificar que el directorio raíz existe
	if _, err := os.Stat(absRoot); err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		Root:      absRoot,
		Name:      filepath.Base(absRoot),
		Files:     make([]FileInfo, 0),
		ScannedAt: time.Now(),
	}

	// Construir set de ignorados
	ignoreSet := make(map[string]bool, len(s.ignorePatterns))
	for _, p := range s.ignorePatterns {
		ignoreSet[p] = true
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Si no podemos leer un directorio, saltarlo
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calcular path relativo para comparar con ignore patterns
		relPath, _ := filepath.Rel(absRoot, path)

		if d.IsDir() {
			// Ignorar directorios en el ignoreSet
			baseName := d.Name()
			if ignoreSet[baseName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Verificar que ningún segmento del path esté en ignoreSet
		segments := strings.Split(relPath, string(filepath.Separator))
		for _, seg := range segments {
			if ignoreSet[seg] {
				return nil
			}
		}

		// Límite de archivos
		if s.maxFiles > 0 && len(info.Files) >= s.maxFiles {
			return filepath.SkipAll
		}

		// Obtener info del archivo
		fi, err := d.Info()
		if err != nil {
			return nil
		}

		ext := filepath.Ext(path)
		fHash, _ := hashFile(path)

		fileInfo := FileInfo{
			Path:     relPath,
			Name:     d.Name(),
			Ext:      ext,
			Size:     fi.Size(),
			Hash:     fHash,
			Language: languageFromExt(ext),
		}

		info.Files = append(info.Files, fileInfo)
		info.FileCount++
		info.TotalSize += fi.Size()

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Detectar lenguaje principal del proyecto
	info.Language = s.detectProjectLanguage(absRoot, info.Files)

	return info, nil
}

func (s *ProjectScanner) detectProjectLanguage(root string, files []FileInfo) Language {
	entries, err := os.ReadDir(root)
	if err != nil {
		return languageByExtension(files)
	}

	entryNames := make(map[string]bool, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			entryNames[e.Name()] = true
		}
	}

	// Check por archivos de proyecto conocidos
	if entryNames["go.mod"] {
		return LanguageGo
	}
	if entryNames["Cargo.toml"] {
		return LanguageRust
	}
	if entryNames["tsconfig.json"] || entryNames["deno.json"] {
		return LanguageTypeScript
	}
	if entryNames["package.json"] {
		// Diferenciar TS de JS por la presencia de archivos .ts
		hasTS := false
		for _, f := range files {
			if f.Language == LanguageTypeScript {
				hasTS = true
				break
			}
		}
		if hasTS {
			return LanguageTypeScript
		}
		return LanguageJavaScript
	}
	if entryNames["pyproject.toml"] || entryNames["requirements.txt"] || entryNames["setup.py"] {
		return LanguagePython
	}
	if entryNames["Gemfile"] {
		return LanguageRuby
	}
	if entryNames["composer.json"] {
		return LanguagePHP
	}
	if entryNames["CMakeLists.txt"] {
		// Diferenciar C de C++ por extensión
		hasCPP := false
		for _, f := range files {
			if f.Language == LanguageCpp {
				hasCPP = true
				break
			}
		}
		if hasCPP {
			return LanguageCpp
		}
		return LanguageC
	}
	if entryNames["mix.exs"] {
		return LanguageElixir
	}
	if entryNames["Package.swift"] || entryNames["Package.resolved"] {
		return LanguageSwift
	}

	return languageByExtension(files)
}

func languageByExtension(files []FileInfo) Language {
	counts := make(map[Language]int)
	for _, f := range files {
		if f.Language != LanguageUnknown {
			counts[f.Language]++
		}
	}

	maxLang := LanguageUnknown
	maxCount := 0
	for lang, count := range counts {
		if count > maxCount {
			maxCount = count
			maxLang = lang
		}
	}
	return maxLang
}

func languageFromExt(ext string) Language {
	if lang, ok := extLangMap[ext]; ok {
		return lang
	}
	return LanguageUnknown
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
