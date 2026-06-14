package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config defines all inputs required to scaffold a new project.
type Config struct {
	ProjectName     string
	Language        string
	Module          string
	Problem         string
	SuccessCriteria string
	Version         string
	Source          string
	ScaffoldDir     string
	LaunchOpenCode  bool
	RawHandoff      string
}

// Result describes what was produced by a successful scaffold run.
type Result struct {
	TargetDir        string
	FilesCreated     int
	OpenCodeLaunched bool
}

// Run orchestrates template rendering, file writing, and optional opencode launch.
//
// Flow:
//  1. Normalize the project name
//  2. Determine target directory
//  3. Check target does not exist
//  4. Render all templates via Renderer
//  5. Write project tree via WriteProject
//  6. (opencode launch is handled by the CLI layer)
func Run(cfg Config) (*Result, error) {
	name := normalizeProjectName(cfg.ProjectName)

	targetDir := cfg.ScaffoldDir
	if targetDir == "" {
		targetDir = name
	}

	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("scaffold: directory %q already exists", targetDir)
	}

	module := cfg.Module
	if module == "" {
		module = "github.com/" + name
	}

	renderer := NewRenderer()

	// Build a context Config with normalized fields for templates.
	tmplCfg := cfg
	tmplCfg.ProjectName = name
	tmplCfg.Module = module

	type job struct {
		tmplName string // path inside embedded FS (e.g. "go-project/AGENT.md.tmpl")
		outPath  string // relative output path in target dir
	}

	mainGoPath := filepath.Join("cmd", name, "main.go")

	jobs := []job{
		{tmplName: "templates/go-project/AGENT.md.tmpl", outPath: "AGENT.md"},
		{tmplName: "templates/go-project/opencode.json.tmpl", outPath: "opencode.json"},
		{tmplName: "templates/go-project/handoff.yaml.tmpl", outPath: "handoff.yaml"},
		{tmplName: "templates/go-project/.gitignore.tmpl", outPath: ".gitignore"},
		{tmplName: "templates/go-project/README.md.tmpl", outPath: "README.md"},
		{tmplName: "templates/go-project/main.go.tmpl", outPath: mainGoPath},
	}

	files := make(map[string]string, len(jobs)+7)

	for _, j := range jobs {
		content, err := renderer.Render(j.tmplName, tmplCfg)
		if err != nil {
			return nil, fmt.Errorf("scaffold: %w", err)
		}
		files[j.outPath] = content
	}

	// Script files — raw copies, not templates.
	scriptEntries := []struct{ embedPath, outPath string }{
		{"templates/go-project/scripts/explorer.py", "scripts/explorer.py"},
		{"templates/go-project/scripts/test-runner.py", "scripts/test-runner.py"},
		{"templates/go-project/scripts/linter.py", "scripts/linter.py"},
	}

	for _, se := range scriptEntries {
		content, err := scriptsFS.ReadFile(se.embedPath)
		if err != nil {
			return nil, fmt.Errorf("scaffold: read script %s: %w", se.embedPath, err)
		}
		files[se.outPath] = string(content)
	}

	// Empty directories required by the scaffold structure.
	files["skills/"] = ""
	files["docs/contexto_proyecto/"] = ""
	files["docs/recursos/"] = ""
	files["internal/"] = ""

	if err := WriteProject(targetDir, files); err != nil {
		return nil, err
	}

	return &Result{
		TargetDir:    targetDir,
		FilesCreated: len(jobs) + len(scriptEntries),
	}, nil
}

// normalizeProjectName produces a safe, kebab-case project name.
func normalizeProjectName(name string) string {
	name = strings.ToLower(name)
	name = strings.Join(strings.Fields(name), "-")
	return toKebabCase(name)
}
