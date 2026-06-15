package taskcontext

import (
	"flag"
	"os"
	"testing"

	helix "github.com/secko/zyrocli/internal/db/helix"
)

var update = flag.Bool("update", false, "update golden files")

// fixture returns a TaskContext with sample data for golden file tests.
func fixture() *TaskContext {
	return &TaskContext{
		TaskID: 42,
		Skills: []*helix.Node{
			{ID: 1, Type: "Skill", Properties: map[string]any{"name": "golang"}},
		},
		CodeNodes: []*helix.Node{
			{ID: 2, Type: "CodeNode", Properties: map[string]any{"name": "main.go"}},
		},
		Docs: []*helix.Node{
			{ID: 3, Type: "Doc", Properties: map[string]any{"name": "README.md"}},
		},
		Patterns: []*helix.Node{
			{ID: 4, Type: "Pattern", Properties: map[string]any{"name": "repository"}},
		},
		Dependents: []*helix.Node{
			{ID: 5, Type: "Task", Properties: map[string]any{"name": "auth-module"}},
		},
		Dependencies: []*helix.Node{
			{ID: 6, Type: "Task", Properties: map[string]any{"name": "db-layer"}},
		},
	}
}

func assertGolden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s", path)
	}
}

func TestFormatJSON(t *testing.T) {
	tc := fixture()
	got, err := tc.FormatJSON()
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "testdata/taskcontext_json.golden", got)
}

func TestFormatPrompt(t *testing.T) {
	tc := fixture()
	got := tc.FormatPrompt()
	assertGolden(t, "testdata/taskcontext_prompt.golden", got)
}

func TestFormatText(t *testing.T) {
	tc := fixture()
	got := tc.FormatText()
	assertGolden(t, "testdata/taskcontext_text.golden", got)
}

func TestFormatEmpty(t *testing.T) {
	tc := &TaskContext{TaskID: 1}
	got := tc.FormatPrompt()
	if got == "" {
		t.Fatal("expected non-empty format for empty context")
	}
	gotText := tc.FormatText()
	if gotText == "" {
		t.Fatal("expected non-empty format for empty context")
	}
}
