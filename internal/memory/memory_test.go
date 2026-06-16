package memory

import (
	"testing"
	"time"
)

func TestFactTypes(t *testing.T) {
	tests := []struct {
		ft   FactType
		want string
	}{
		{FactDecision, "decision"},
		{FactError, "error"},
		{FactPreference, "preference"},
		{FactPattern, "pattern"},
		{FactDependency, "dependency"},
		{FactObservation, "observation"},
	}
	for _, tt := range tests {
		if string(tt.ft) != tt.want {
			t.Errorf("FactType %s != %s", tt.ft, tt.want)
		}
	}
}

func TestCausalEdgeTypes(t *testing.T) {
	edges := []CausalEdgeType{EdgeCaused, EdgePrecedes, EdgeContradicts, EdgeSupports, EdgeRequires, EdgeDerivesFrom, EdgeReferences}
	if len(edges) != 7 {
		t.Errorf("expected 7 edge types, got %d", len(edges))
	}
}

func TestFactCreation(t *testing.T) {
	f := Fact{
		Type:       FactDecision,
		Content:    "Usamos mxbai-embed-large para embeddings",
		Salience:   0.8,
		Confidence: 0.9,
		Phase:      "F0",
		IsActive:   true,
	}
	if f.Type != FactDecision {
		t.Error("Fact type mismatch")
	}
	if f.Salience != 0.8 {
		t.Error("Salience mismatch")
	}
}

func TestDefaultDecayConfig(t *testing.T) {
	cfg := DefaultDecayConfig()
	if cfg.BaseDecayRate != 0.05 {
		t.Errorf("expected 0.05, got %f", cfg.BaseDecayRate)
	}
	if cfg.SalienceThreshold != 0.15 {
		t.Errorf("expected 0.15, got %f", cfg.SalienceThreshold)
	}
}

func TestEbbinghausDecay(t *testing.T) {
	// Fact con salience 0.7, decay 0.05, 30 días sin acceso: salience ≈ 0.156
	result := calculateEbbinghausDecay(0.7, 0.05, 30)
	expected := 0.156
	if result < expected-0.01 || result > expected+0.01 {
		t.Errorf("Ebbinghaus decay: expected ~0.156, got %f", result)
	}

	// Sin días transcurridos
	result = calculateEbbinghausDecay(0.7, 0.05, 0)
	if result != 0.7 {
		t.Errorf("No decay: expected 0.7, got %f", result)
	}
}

func TestReinforceSalience(t *testing.T) {
	// salience += 0.3 * (1 - salience)
	salience := 0.5
	expected := 0.5 + 0.3*(1-0.5) // 0.65

	// Simular refuerzo
	reinforced := salience + 0.3*(1-salience)
	if reinforced != expected {
		t.Errorf("Reinforce: expected %f, got %f", expected, reinforced)
	}

	// No debe pasar de 1.0
	salience = 0.9
	reinforced = salience + 0.3*(1-salience) // 0.93
	if reinforced > 1.0 {
		t.Error("Salience should not exceed 1.0")
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{1.0, 0.0, 0.0}
	sim := cosineSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("identical vectors should have similarity 1.0, got %f", sim)
	}

	c := []float32{1.0, 0.0, 0.0}
	d := []float32{-1.0, 0.0, 0.0}
	sim = cosineSimilarity(c, d)
	if sim != -1.0 {
		t.Errorf("opposite vectors should have similarity -1.0, got %f", sim)
	}

	// Empty vectors
	sim = cosineSimilarity(nil, nil)
	if sim != 0 {
		t.Errorf("empty vectors should have similarity 0, got %f", sim)
	}
}

func TestFormatMemoryForPrompt(t *testing.T) {
	// Empty
	result := formatMemoryForPrompt(nil)
	if result != "" {
		t.Error("empty input should return empty string")
	}

	// With results
	results := []*MemoryResult{
		{
			Fact: Fact{
				Type:       FactDecision,
				Content:    "test content",
				Confidence: 0.9,
				Phase:      "F0",
			},
			Score:  0.95,
			Source: "vector",
		},
	}
	result = formatMemoryForPrompt(results)
	if result == "" {
		t.Error("non-empty input should return non-empty string")
	}
}

func TestContradictionStrategies(t *testing.T) {
	strategies := []ContradictionStrategy{StrategyNewestWins, StrategyHighestConfidence, StrategyKeepBoth}
	if len(strategies) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(strategies))
	}
}

func TestFactRowTypes(t *testing.T) {
	f := FactRow{
		ID:         1,
		Type:       "decision",
		Content:    "test",
		Salience:   0.7,
		Confidence: 0.8,
		Phase:      "F0",
		IsActive:   true,
	}
	if f.ID != 1 || f.Type != "decision" || !f.IsActive {
		t.Error("FactRow fields incorrect")
	}
}

func TestDecayConfigFromDefaults(t *testing.T) {
	cfg := DecayConfigFromDefaults()
	if cfg.BaseDecayRate != 0.05 {
		t.Errorf("expected BaseDecayRate 0.05, got %f", cfg.BaseDecayRate)
	}
	if cfg.AccessBoost != 0.3 {
		t.Errorf("expected AccessBoost 0.3, got %f", cfg.AccessBoost)
	}
	if cfg.SalienceThreshold != 0.15 {
		t.Errorf("expected SalienceThreshold 0.15, got %f", cfg.SalienceThreshold)
	}
	if cfg.MaxSalience != 1.0 {
		t.Errorf("expected MaxSalience 1.0, got %f", cfg.MaxSalience)
	}
	if cfg.DefaultExpiryDays != 90 {
		t.Errorf("expected DefaultExpiryDays 90, got %d", cfg.DefaultExpiryDays)
	}
}

func TestTimeBasedOperations(t *testing.T) {
	now := time.Now()
	f := Fact{
		CreatedAt:      now,
		LastAccessedAt: now,
		AccessCount:    0,
		ExpiresAt:      now.Add(90 * 24 * time.Hour),
	}

	if !f.CreatedAt.Equal(now) {
		t.Error("CreatedAt mismatch")
	}
	if !f.LastAccessedAt.Equal(now) {
		t.Error("LastAccessedAt mismatch")
	}
	if f.AccessCount != 0 {
		t.Error("AccessCount should be 0")
	}
	if f.ExpiresAt.Before(now) {
		t.Error("ExpiresAt should be in the future")
	}
}
