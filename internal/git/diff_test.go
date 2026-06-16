package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseDiffOutput — unit tests
// ---------------------------------------------------------------------------

func TestParseDiffOutput_Normal(t *testing.T) {
	output := "M\tmain.go\nA\tinternal/new.go\nD\told.go"

	files := parseDiffOutput(output)
	require.Len(t, files, 3)

	assert.Equal(t, "M", files[0].Status)
	assert.Equal(t, "main.go", files[0].Path)
	assert.Empty(t, files[0].OldPath)

	assert.Equal(t, "A", files[1].Status)
	assert.Equal(t, "internal/new.go", files[1].Path)

	assert.Equal(t, "D", files[2].Status)
	assert.Equal(t, "old.go", files[2].Path)
}

func TestParseDiffOutput_Rename(t *testing.T) {
	output := "R100\told.go\tnew.go"

	files := parseDiffOutput(output)
	require.Len(t, files, 1)

	assert.Equal(t, "R100", files[0].Status)
	assert.Equal(t, "new.go", files[0].Path)
	assert.Equal(t, "old.go", files[0].OldPath)
}

func TestParseDiffOutput_Empty(t *testing.T) {
	files := parseDiffOutput("")
	assert.Nil(t, files)

	files = parseDiffOutput("  ")
	assert.Nil(t, files)
}

func TestParseDiffOutput_Multiple(t *testing.T) {
	output := "M\tfile1.go\nA\tfile2.go\nR090\told_name.go\tnew_name.go\nD\tfile3.go\nM\tfile4.go"

	files := parseDiffOutput(output)
	require.Len(t, files, 5)

	assert.Equal(t, "M", files[0].Status)
	assert.Equal(t, "file1.go", files[0].Path)

	assert.Equal(t, "A", files[1].Status)
	assert.Equal(t, "file2.go", files[1].Path)

	assert.Equal(t, "R090", files[2].Status)
	assert.Equal(t, "new_name.go", files[2].Path)
	assert.Equal(t, "old_name.go", files[2].OldPath)

	assert.Equal(t, "D", files[3].Status)
	assert.Equal(t, "file3.go", files[3].Path)

	assert.Equal(t, "M", files[4].Status)
	assert.Equal(t, "file4.go", files[4].Path)
}

func TestParseDiffOutput_MalformedLine(t *testing.T) {
	// Lines with only status but no path should be skipped
	output := "M\nA\tsome.go"

	files := parseDiffOutput(output)
	require.Len(t, files, 1)
	assert.Equal(t, "A", files[0].Status)
	assert.Equal(t, "some.go", files[0].Path)
}

// ---------------------------------------------------------------------------
// ChangedFile helper methods
// ---------------------------------------------------------------------------

func TestChangedFile_IsModified(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"modified", "M", true},
		{"added", "A", true},
		{"deleted", "D", false},
		{"renamed", "R100", false},
		{"unknown", "X", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := ChangedFile{Status: tt.status}
			assert.Equal(t, tt.want, cf.IsModified())
		})
	}
}

func TestChangedFile_IsRename(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"renamed 100", "R100", true},
		{"renamed 90", "R090", true},
		{"modified", "M", false},
		{"added", "A", false},
		{"deleted", "D", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := ChangedFile{Status: tt.status}
			assert.Equal(t, tt.want, cf.IsRename())
		})
	}
}

func TestChangedFile_IsDeleted(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"deleted", "D", true},
		{"modified", "M", false},
		{"added", "A", false},
		{"renamed", "R100", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := ChangedFile{Status: tt.status}
			assert.Equal(t, tt.want, cf.IsDeleted())
		})
	}
}

// ---------------------------------------------------------------------------
// ChangedFiles — integration tests with temp git repo
// ---------------------------------------------------------------------------

func TestChangedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}

	dir := t.TempDir()

	// Init git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	// Create initial file and commit
	writeFile(t, dir, "README.md", "# Initial")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial commit")

	// Modify file
	writeFile(t, dir, "README.md", "# Modified")
	writeFile(t, dir, "new.go", "package main")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "second commit")

	// ChangedFiles against HEAD~1
	files, err := ChangedFiles("HEAD~1", dir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	// Find and verify each file
	var readmeFound, newGoFound bool
	for _, f := range files {
		switch f.Path {
		case "README.md":
			assert.Equal(t, "M", f.Status)
			readmeFound = true
		case "new.go":
			assert.Equal(t, "A", f.Status)
			newGoFound = true
		}
	}
	assert.True(t, readmeFound, "README.md should be in changed files")
	assert.True(t, newGoFound, "new.go should be in changed files")
}

func TestChangedFiles_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}

	dir := t.TempDir()

	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	writeFile(t, dir, "keep.go", "package main\n")
	writeFile(t, dir, "remove.go", "package main\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial")

	// Delete one file
	os.Remove(filepath.Join(dir, "remove.go"))
	writeFile(t, dir, "keep.go", "package main\n// updated\n")
	runCmd(t, dir, "git", "add", "-A")
	runCmd(t, dir, "git", "commit", "-m", "delete remove.go")

	files, err := ChangedFiles("HEAD~1", dir)
	require.NoError(t, err)

	var foundDelete bool
	for _, f := range files {
		if f.Path == "remove.go" {
			assert.True(t, f.IsDeleted())
			foundDelete = true
		}
	}
	assert.True(t, foundDelete, "remove.go should appear as deleted")
}

func TestChangedFiles_Rename(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}

	dir := t.TempDir()

	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	writeFile(t, dir, "old.go", "package main\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial")

	// Rename
	os.Rename(filepath.Join(dir, "old.go"), filepath.Join(dir, "new.go"))
	runCmd(t, dir, "git", "add", "-A")
	runCmd(t, dir, "git", "commit", "-m", "rename old.go to new.go")

	files, err := ChangedFiles("HEAD~1", dir)
	require.NoError(t, err)

	var foundRename bool
	for _, f := range files {
		if f.Path == "new.go" && f.OldPath == "old.go" {
			assert.True(t, f.IsRename())
			foundRename = true
		}
	}
	assert.True(t, foundRename, "rename should be detected")
}

func TestChangedFiles_NoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}

	dir := t.TempDir()

	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	writeFile(t, dir, "stable.go", "package main\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial")

	// Same ref — no changes
	files, err := ChangedFiles("HEAD", dir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestChangedFiles_NoGitDir(t *testing.T) {
	dir := t.TempDir() // no git init

	_, err := ChangedFiles("HEAD", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git diff")
}

func TestChangedFilesBetween(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in -short mode")
	}

	dir := t.TempDir()

	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	writeFile(t, dir, "v1.go", "package main\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "v1")

	writeFile(t, dir, "v2.go", "package main\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "v2")

	files, err := ChangedFilesBetween("HEAD~1", "HEAD", dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "v2.go", files[0].Path)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, string(out))
}

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(t, err)
}
