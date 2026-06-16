package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create .zyro directory
	if err := os.MkdirAll(filepath.Join(dir, ".zyro"), 0o755); err != nil {
		t.Fatalf("mkdir .zyro: %v", err)
	}
	return dir
}

func writeConventions(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".zyro", "conventions.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write conventions: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GenerateIndex tests
// ---------------------------------------------------------------------------

func TestGenerateIndex_EmptyConventions(t *testing.T) {
	dir := tempDir(t)
	writeConventions(t, dir, "topic_keys:\n  project: {}\n  change: {}\n  graph: {}\n")

	idx, err := GenerateIndex(dir)
	if err != nil {
		t.Fatalf("GenerateIndex failed: %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(idx.Entries))
	}
}

func TestGenerateIndex_WithProjectKeys(t *testing.T) {
	dir := tempDir(t)
	writeConventions(t, dir, `
topic_keys:
  project:
    context: "zyro/testproj/context"
    doc_index: "zyro/testproj/doc-index"
    architecture: "zyro/testproj/architecture"
    changelog: "zyro/testproj/changelog"
  change: {}
  graph: {}
`)

	idx, err := GenerateIndex(dir)
	if err != nil {
		t.Fatalf("GenerateIndex failed: %v", err)
	}
	// 4 project keys: context, doc_index, architecture, changelog
	if len(idx.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d: %+v", len(idx.Entries), idx.Entries)
	}

	// Verify specific entries exist
	topicKeys := make(map[string]bool)
	for _, e := range idx.Entries {
		topicKeys[e.TopicKey] = true
	}

	expectedKeys := []string{
		"zyro/testproj/context",
		"zyro/testproj/doc-index",
		"zyro/testproj/architecture",
	}
	for _, k := range expectedKeys {
		if !topicKeys[k] {
			t.Errorf("expected topic_key %q not found", k)
		}
	}
}

func TestGenerateIndex_WithChanges(t *testing.T) {
	dir := tempDir(t)
	writeConventions(t, dir, `
topic_keys:
  project:
    context: "zyro/testproj/context"
  change:
    explore: "sdd/{change}/explore"
    proposal: "sdd/{change}/proposal"
  graph: {}
`)

	// Create an active change directory
	changesDir := filepath.Join(dir, "openspec", "changes", "my-change")
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatalf("mkdir changes: %v", err)
	}

	idx, err := GenerateIndex(dir)
	if err != nil {
		t.Fatalf("GenerateIndex failed: %v", err)
	}

	// Should have 1 project entry + 2 change entries
	expectedCount := 1 + 2
	if len(idx.Entries) != expectedCount {
		t.Errorf("expected %d entries, got %d", expectedCount, len(idx.Entries))
	}

	// Check change entries
	var changeEntries int
	for _, e := range idx.Entries {
		if e.ChangeName == "my-change" {
			changeEntries++
			if e.Type != "change/explore" && e.Type != "change/proposal" {
				t.Errorf("unexpected change entry type: %s", e.Type)
			}
		}
	}
	if changeEntries != 2 {
		t.Errorf("expected 2 change entries, got %d", changeEntries)
	}
}

func TestGenerateIndex_MissingConventions(t *testing.T) {
	dir := tempDir(t)
	// No conventions.yaml written

	_, err := GenerateIndex(dir)
	if err == nil {
		t.Fatal("expected error for missing conventions.yaml")
	}
	if !strings.Contains(err.Error(), "conventions") {
		t.Errorf("expected error about conventions, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SaveIndex / LoadIndex tests
// ---------------------------------------------------------------------------

func TestSaveAndLoadIndex(t *testing.T) {
	dir := tempDir(t)
	original := &DocIndex{
		GeneratedAt: "2026-01-01T00:00:00Z",
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/context", Type: "project/context", ObservationID: 1, LastModified: "2026-01-01"},
			{TopicKey: "sdd/test-change/explore", Type: "change/explore", ChangeName: "test-change"},
		},
	}

	if err := SaveIndex(dir, original); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Check the file exists
	indexPath := filepath.Join(dir, ".zyro", "doc-index.yaml")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index file not created: %v", err)
	}

	loaded, err := LoadIndex(dir)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}
	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].TopicKey != "zyro/test/context" {
		t.Errorf("expected topic_key 'zyro/test/context', got %q", loaded.Entries[0].TopicKey)
	}
}

func TestLoadIndex_NotFound(t *testing.T) {
	dir := tempDir(t)
	_, err := LoadIndex(dir)
	if err == nil {
		t.Fatal("expected error for missing index")
	}
}

// ---------------------------------------------------------------------------
// SearchIndex tests
// ---------------------------------------------------------------------------

func TestSearchIndex_ExactTopicKey(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/context", Type: "project/context"},
			{TopicKey: "sdd/change-x/explore", Type: "change/explore", ChangeName: "change-x"},
		},
	}

	results, err := SearchIndex(idx, SearchQuery{TopicKey: "zyro/test/context"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Entry.TopicKey != "zyro/test/context" {
		t.Errorf("expected 'zyro/test/context', got %q", results[0].Entry.TopicKey)
	}
}

func TestSearchIndex_ExactTopicKeyNotFound(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/context", Type: "project/context"},
		},
	}

	_, err := SearchIndex(idx, SearchQuery{TopicKey: "zyro/nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing topic key")
	}
}

func TestSearchIndex_ByQuery(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "sdd/change-x/explore", Type: "change/explore", ChangeName: "change-x"},
			{TopicKey: "sdd/change-y/proposal", Type: "change/proposal", ChangeName: "change-y"},
			{TopicKey: "zyro/proj/context", Type: "project/context"},
		},
	}

	results, err := SearchIndex(idx, SearchQuery{Query: "change-x"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'change-x', got %d", len(results))
	}
}

func TestSearchIndex_FallbackByType(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/proj/context", Type: "project/context"},
			{TopicKey: "zyro/proj/doc-index", Type: "project/doc-index"},
		},
	}

	results, err := SearchIndex(idx, SearchQuery{Type: "project"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSearchIndex_FallbackByChange(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "sdd/c1/explore", Type: "change/explore", ChangeName: "c1"},
			{TopicKey: "sdd/c1/proposal", Type: "change/proposal", ChangeName: "c1"},
			{TopicKey: "sdd/c2/explore", Type: "change/explore", ChangeName: "c2"},
			{TopicKey: "zyro/proj/context", Type: "project/context"},
		},
	}

	results, err := SearchIndex(idx, SearchQuery{ChangeName: "c1"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for change 'c1', got %d", len(results))
	}
}

func TestSearchIndex_EmptyIndex(t *testing.T) {
	idx := &DocIndex{}

	results, err := SearchIndex(idx, SearchQuery{Type: "project"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchIndex_LastResort(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/proj/context", Type: "project/context"},
		},
	}

	results, err := SearchIndex(idx, SearchQuery{Query: "nonexistent"})
	if err != nil {
		t.Fatalf("SearchIndex failed: %v", err)
	}
	// Query didn't match, but Type/ChangeName are empty, so fallback returns all
	// This is the correct "last resort" behavior: return what we have
	if len(results) == 0 {
		t.Error("expected at least fallback results")
	}
}

func TestSearchIndex_NilIndex(t *testing.T) {
	_, err := SearchIndex(nil, SearchQuery{})
	if err == nil {
		t.Fatal("expected error for nil index")
	}
}

// ---------------------------------------------------------------------------
// MustFindByTopicKey tests
// ---------------------------------------------------------------------------

func TestMustFindByTopicKey_Found(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/key", Type: "project/test"},
		},
	}

	result := MustFindByTopicKey(idx, "zyro/test/key")
	if result.Entry.TopicKey != "zyro/test/key" {
		t.Errorf("expected topic_key 'zyro/test/key', got %q", result.Entry.TopicKey)
	}
}

func TestMustFindByTopicKey_Panics(t *testing.T) {
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/key", Type: "project/test"},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing key")
		}
	}()

	MustFindByTopicKey(idx, "zyro/missing")
}

// ---------------------------------------------------------------------------
// Sync tests
// ---------------------------------------------------------------------------

func TestSync_FullCycle(t *testing.T) {
	dir := tempDir(t)
	writeConventions(t, dir, `
topic_keys:
  project:
    context: "zyro/testproj/context"
    doc_index: "zyro/testproj/doc-index"
    architecture: "zyro/testproj/architecture"
    changelog: "zyro/testproj/changelog"
  change: {}
  graph: {}
`)

	idx, err := Sync(dir)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index from sync")
	}
	if len(idx.Entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(idx.Entries))
	}

	// Verify that index file was written
	if _, err := os.Stat(filepath.Join(dir, ".zyro", "doc-index.yaml")); err != nil {
		t.Errorf("doc-index.yaml not written: %v", err)
	}

	// Verify that ARCHITECTURE.md was written
	if _, err := os.Stat(filepath.Join(dir, "ARCHITECTURE.md")); err != nil {
		t.Errorf("ARCHITECTURE.md not written: %v", err)
	}

	// Verify that CHANGELOG.md was written
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); err != nil {
		t.Errorf("CHANGELOG.md not written: %v", err)
	}

	// Verify graph state was written
	if _, err := os.Stat(filepath.Join(dir, ".zyro", "graph-state.yaml")); err != nil {
		t.Errorf("graph-state.yaml not written: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Graphify tests
// ---------------------------------------------------------------------------

func TestUpdateGraph_FirstRun(t *testing.T) {
	dir := tempDir(t)
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/k1", Type: "project/test"},
			{TopicKey: "zyro/test/k2", Type: "project/test"},
		},
	}

	if err := UpdateGraph(dir, idx); err != nil {
		t.Fatalf("UpdateGraph failed: %v", err)
	}

	// Check state file
	statePath := filepath.Join(dir, ".zyro", "graph-state.yaml")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("cannot read graph state: %v", err)
	}
	if !strings.Contains(string(data), "entry_count: 2") {
		t.Errorf("expected entry_count 2, got: %s", string(data))
	}
}

func TestUpdateGraph_NoSignificantChange(t *testing.T) {
	dir := tempDir(t)

	// Write previous state with 2 entries
	statePath := filepath.Join(dir, ".zyro", "graph-state.yaml")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("entry_count: 2\nchecksum: 2\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	// Current index has 3 entries — diff=1, threshold=5, not significant
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/k1", Type: "project/test"},
			{TopicKey: "zyro/test/k2", Type: "project/test"},
			{TopicKey: "zyro/test/k3", Type: "project/test"},
		},
	}

	if err := UpdateGraph(dir, idx); err != nil {
		t.Fatalf("UpdateGraph failed: %v", err)
	}

	// State should NOT have been updated (diff < threshold)
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("cannot read graph state: %v", err)
	}
	if !strings.Contains(string(data), "entry_count: 2") {
		t.Errorf("expected state to remain at 2 entries, got: %s", string(data))
	}
}

func TestUpdateGraph_SignificantChange(t *testing.T) {
	dir := tempDir(t)

	// Write previous state with 1 entry
	statePath := filepath.Join(dir, ".zyro", "graph-state.yaml")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("entry_count: 1\nchecksum: 1\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	// Current index has 10 entries — diff=9, threshold=5, significant
	idx := &DocIndex{
		Entries: make([]IndexEntry, 10),
	}

	if err := UpdateGraph(dir, idx); err != nil {
		t.Fatalf("UpdateGraph failed: %v", err)
	}

	// State should have been updated
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("cannot read graph state: %v", err)
	}
	if !strings.Contains(string(data), "entry_count: 10") {
		t.Errorf("expected state to update to 10 entries, got: %s", string(data))
	}
}

func TestUpdateGraph_NilIndex(t *testing.T) {
	dir := tempDir(t)
	err := UpdateGraph(dir, nil)
	if err == nil {
		t.Fatal("expected error for nil index")
	}
}

// ---------------------------------------------------------------------------
// Export tests
// ---------------------------------------------------------------------------

func TestExport_GeneratesFiles(t *testing.T) {
	dir := tempDir(t)
	idx := &DocIndex{
		Entries: []IndexEntry{
			{TopicKey: "zyro/test/context", Type: "project/context"},
			{TopicKey: "sdd/test-change/explore", Type: "change/explore", ChangeName: "test-change"},
		},
	}

	if err := Export(dir, idx); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check ARCHITECTURE.md
	archPath := filepath.Join(dir, "ARCHITECTURE.md")
	archData, err := os.ReadFile(archPath)
	if err != nil {
		t.Fatalf("ARCHITECTURE.md not found: %v", err)
	}
	if !strings.Contains(string(archData), "zyro/test/context") {
		t.Errorf("ARCHITECTURE.md missing topic key: %s", string(archData))
	}
	if !strings.Contains(string(archData), "sdd/test-change/explore") {
		t.Errorf("ARCHITECTURE.md missing change topic key: %s", string(archData))
	}

	// Check CHANGELOG.md
	chgPath := filepath.Join(dir, "CHANGELOG.md")
	chgData, err := os.ReadFile(chgPath)
	if err != nil {
		t.Fatalf("CHANGELOG.md not found: %v", err)
	}
	if !strings.Contains(string(chgData), "test-change") {
		t.Errorf("CHANGELOG.md missing change name: %s", string(chgData))
	}
	if !strings.Contains(string(chgData), "sdd/test-change/explore") {
		t.Errorf("CHANGELOG.md missing topic key: %s", string(chgData))
	}
}

func TestExport_NilIndex(t *testing.T) {
	dir := tempDir(t)
	err := Export(dir, nil)
	if err == nil {
		t.Fatal("expected error for nil index")
	}
}
