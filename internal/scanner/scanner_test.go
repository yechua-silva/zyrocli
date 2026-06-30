package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanner(t *testing.T) {
	s := NewScanner()
	assert.NotNil(t, s)
	assert.Equal(t, 10000, s.maxFiles)
	assert.Equal(t, DefaultIgnorePatterns, s.ignorePatterns)
}

func TestWithIgnorePatterns(t *testing.T) {
	s := NewScanner().WithIgnorePatterns([]string{".custom"})
	assert.Contains(t, s.ignorePatterns, ".custom")
}

func TestWithMaxFiles(t *testing.T) {
	s := NewScanner().WithMaxFiles(100)
	assert.Equal(t, 100, s.maxFiles)
}

func TestHashFile(t *testing.T) {
	// Crear archivo temporal
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("hello"), 0644)
	require.NoError(t, err)

	hash, err := hashFile(tmpFile)
	require.NoError(t, err)
	// SHA256 de "hello"
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", hash)
}

func TestLanguageFromExt(t *testing.T) {
	tests := []struct {
		ext  string
		want Language
	}{
		{".go", LanguageGo},
		{".ts", LanguageTypeScript},
		{".tsx", LanguageTypeScript},
		{".js", LanguageJavaScript},
		{".rs", LanguageRust},
		{".py", LanguagePython},
		{".java", LanguageJava},
		{".unknown", LanguageUnknown},
		{"", LanguageUnknown},
	}
	for _, tt := range tests {
		got := languageFromExt(tt.ext)
		assert.Equal(t, tt.want, got, "languageFromExt(%q)", tt.ext)
	}
}

func TestScanBasic(t *testing.T) {
	// Crear estructura de proyecto temporal
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "cmd", "app"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "cmd", "app", "main.go"), []byte("package main\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test\n"), 0644))

	s := NewScanner()
	info, err := s.Scan(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "Go", string(info.Language))
	assert.Equal(t, 3, info.FileCount)
	assert.Equal(t, filepath.Base(tmpDir), info.Name)
	assert.Equal(t, tmpDir, info.Root)
}

func TestScanIgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg", "index.js"), []byte("// ignored\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.js"), []byte("// kept\n"), 0644))

	s := NewScanner()
	info, err := s.Scan(tmpDir)
	require.NoError(t, err)

	// node_modules debe estar ignorado
	for _, f := range info.Files {
		assert.NotContains(t, f.Path, "node_modules", "node_modules should be ignored")
	}
	assert.Equal(t, 1, info.FileCount, "only src/main.js should be found")
}

func TestScanMaxFiles(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte(name), 0644))
	}

	s := NewScanner().WithMaxFiles(3)
	info, err := s.Scan(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 3, info.FileCount, "should be limited to 3 files")
}

func TestDetectLanguageByProjectFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // path → content
		expected Language
	}{
		{
			name: "go",
			files: map[string]string{
				"go.mod": "module test\n",
				"main.go": "package main\n",
			},
			expected: LanguageGo,
		},
		{
			name: "python",
			files: map[string]string{
				"requirements.txt": "flask\n",
				"app.py": "print('hello')\n",
			},
			expected: LanguagePython,
		},
		{
			name: "rust",
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"test\"\n",
				"main.rs": "fn main() {}\n",
			},
			expected: LanguageRust,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for path, content := range tt.files {
				dir := filepath.Dir(filepath.Join(tmpDir, path))
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0644))
			}

			s := NewScanner()
			info, err := s.Scan(tmpDir)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, info.Language, "language detection for %s", tt.name)
		})
	}
}

func TestScanNonExistentPath(t *testing.T) {
	s := NewScanner()
	_, err := s.Scan("/tmp/nonexistent-path-12345")
	assert.Error(t, err)
}

func TestHashConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	require.NoError(t, os.WriteFile(f1, []byte("same content"), 0644))
	require.NoError(t, os.WriteFile(f2, []byte("same content"), 0644))

	h1, err := hashFile(f1)
	require.NoError(t, err)
	h2, err := hashFile(f2)
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "same content should produce same hash")
}

func TestLanguageByExtension(t *testing.T) {
	files := []FileInfo{
		{Path: "a.go", Language: LanguageGo},
		{Path: "b.go", Language: LanguageGo},
		{Path: "c.py", Language: LanguagePython},
	}
	lang := languageByExtension(files)
	assert.Equal(t, LanguageGo, lang, "Go should win (2 of 3)")
}
