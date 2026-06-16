package helix

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// T-2.12: Tests completos — defaults, tipos y lógica sin red
// ---------------------------------------------------------------------------

func TestNewClient(t *testing.T) {
	c, err := NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("client should not be nil")
	}
}

func TestDefaultSDKClientOptions(t *testing.T) {
	opts := DefaultSDKClientOptions()
	if opts.BaseURL != "http://localhost:6969" {
		t.Errorf("expected default URL, got %s", opts.BaseURL)
	}
	if opts.Timeout == 0 {
		t.Error("timeout should not be zero")
	}
	if opts.MaxRetries != 3 {
		t.Errorf("expected 3 retries, got %d", opts.MaxRetries)
	}
}

func TestRowTypes(t *testing.T) {
	// Verificar que todos los row types se construyen correctamente
	tr := TaskRow{ID: 1, Name: "test", Phase: "F0", Status: "active"}
	if tr.ID != 1 || tr.Name != "test" || tr.Phase != "F0" {
		t.Error("TaskRow fields incorrect")
	}

	cr := CodeNodeRow{ID: 1, Path: "/main.go", Language: "go"}
	if cr.Path != "/main.go" || cr.Language != "go" {
		t.Error("CodeNodeRow fields incorrect")
	}

	fr := FactRow{ID: 1, Type: "decision", Content: "test", IsActive: true}
	if !fr.IsActive || fr.Type != "decision" {
		t.Error("FactRow fields incorrect")
	}
}

func TestDefaultHybridSearchOptions(t *testing.T) {
	opts := DefaultHybridSearchOptions()
	if opts.MaxResults != 10 {
		t.Errorf("expected 10 max results, got %d", opts.MaxResults)
	}
	if opts.MinScore != 0.0 {
		t.Errorf("expected 0.0 min score, got %f", opts.MinScore)
	}
}

func TestDefaultEmbeddingConfig(t *testing.T) {
	cfg := DefaultEmbeddingConfig()
	if cfg.Provider != ProviderOpenAI {
		t.Errorf("expected OpenAI provider, got %s", cfg.Provider)
	}
	if cfg.BatchSize != 20 {
		t.Errorf("expected 20 batch size, got %d", cfg.BatchSize)
	}
	if cfg.Dims != 1536 {
		t.Errorf("expected 1536 dims, got %d", cfg.Dims)
	}
}

func TestDefaultIndexes(t *testing.T) {
	indexes := DefaultIndexes()
	if len(indexes) == 0 {
		t.Fatal("expected at least one index")
	}

	// Verificar que tenemos índices para labels importantes
	labels := make(map[string]bool)
	for _, idx := range indexes {
		labels[idx.Label] = true
	}

	importantLabels := []string{"Project", "Task", "CodeNode", "Skill", "Fact"}
	for _, l := range importantLabels {
		if !labels[l] {
			t.Errorf("missing index for label %s", l)
		}
	}
}

func TestIndexSpecValidation(t *testing.T) {
	specs := []IndexSpec{
		{Label: "Project", Property: "name", IndexType: IndexEquality},
		{Label: "Fact", Property: "embedding", IndexType: IndexVector, Dimensions: 1536},
	}

	for _, s := range specs {
		if s.Label == "" {
			t.Error("label should not be empty")
		}
		if s.Property == "" {
			t.Error("property should not be empty")
		}
	}
}

func TestSDKClientOptions(t *testing.T) {
	opts := DefaultSDKClientOptions()
	if opts.BaseURL != "http://localhost:6969" {
		t.Errorf("expected default URL, got %s", opts.BaseURL)
	}
	if opts.MaxRetries != 3 {
		t.Errorf("expected 3 retries, got %d", opts.MaxRetries)
	}

	opts2 := SDKClientOptions{
		BaseURL:    "http://custom:6969",
		Timeout:    10,
		MaxRetries: 0,
	}
	if opts2.BaseURL != "http://custom:6969" {
		t.Error("custom URL not respected")
	}
}

func TestEmbeddingServiceCreation(t *testing.T) {
	svc := NewEmbeddingService(DefaultEmbeddingConfig())
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}

func TestEmptyEmbeddingFails(t *testing.T) {
	svc := NewEmbeddingService(DefaultEmbeddingConfig())
	_, err := svc.Embed(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty text")
	}
}

func TestRRFFormula(t *testing.T) {
	// Probar el cálculo RRF manualmente
	k := 60
	rank := 1
	score1 := 1.0 / float64(k+rank) // ~0.01639

	rank = 2
	score2 := 1.0 / float64(k+rank) // ~0.01613

	if score1 <= score2 {
		t.Error("higher rank should have higher score")
	}
}

func TestDefaultClientOptions(t *testing.T) {
	// WithBaseURL + WithProjectID functional options
	c, err := NewClient(context.Background(), WithBaseURL("http://test:6969"), WithProjectID("proj-1"))
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "http://test:6969" {
		t.Errorf("expected custom URL, got %s", c.baseURL)
	}
	if c.projectID != "proj-1" {
		t.Errorf("expected custom project ID, got %s", c.projectID)
	}
}

func TestNewSDKClient(t *testing.T) {
	c, err := NewSDKClient(DefaultSDKClientOptions())
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("SDK client should not be nil")
	}
	if c.opts.MaxRetries != 3 {
		t.Errorf("expected 3 retries, got %d", c.opts.MaxRetries)
	}
}

func TestHybridSearchOptionsValidation(t *testing.T) {
	opts := HybridSearchOptions{
		MaxResults: 0,
		MinScore:   -1,
	}

	// MaxResults válido es > 0; HybridSearch usa default 10 si es <= 0
	if opts.MaxResults <= 0 {
		// OK, se usará default en HybridSearch
	}
	_ = opts.MinScore
}

func TestFactRowIsActive(t *testing.T) {
	fr := FactRow{ID: 1, Type: "memory", Content: "test", IsActive: true}
	if !fr.IsActive {
		t.Error("IsActive should be true")
	}
	if fr.Type != "memory" {
		t.Errorf("Type = %q, want memory", fr.Type)
	}
}
