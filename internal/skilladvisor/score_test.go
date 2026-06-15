package skilladvisor

import (
	"testing"
)

func zeroAlerts() *int { z := 0; return &z }

func TestScoreSkill_exactMatch(t *testing.T) {
	s := Skill{Language: "go", Framework: "cobra", ProjectType: "cli", SocketAlerts: zeroAlerts(), GenAgentTrustHub: "Safe"}
	ctx := ProjectContext{Language: "go", Framework: "cobra", ProjectType: "cli"}
	score := ScoreSkill(s, ctx)
	if score != 100 {
		t.Errorf("exact match: expected 100 (10+20+30+25+15), got %d", score)
	}
}

func TestScoreSkill_partialMatch(t *testing.T) {
	s := Skill{Language: "go", ProjectType: "cli"}
	ctx := ProjectContext{Language: "go", Framework: "react", ProjectType: "web"}
	score := ScoreSkill(s, ctx)
	if score != 10 {
		t.Errorf("partial match (language only): expected 10, got %d", score)
	}
}

func TestScoreSkill_noMatch(t *testing.T) {
	s := Skill{Language: "python", ProjectType: "api"}
	ctx := ProjectContext{Language: "go", Framework: "cobra", ProjectType: "cli"}
	score := ScoreSkill(s, ctx)
	if score != 0 {
		t.Errorf("no match: expected 0, got %d", score)
	}
}

func TestScoreSkill_verifiedPublisher(t *testing.T) {
	s := Skill{Language: "go", Publisher: "anthropics", ProjectType: "cli", SocketAlerts: zeroAlerts(), GenAgentTrustHub: "Safe"}
	ctx := ProjectContext{Language: "go", ProjectType: "cli"}
	score := ScoreSkill(s, ctx)
	expected := 10 + 30 + 50 + 25 + 15 // language + project + publisher + genagent + socket
	if score != expected {
		t.Errorf("verified publisher: expected %d, got %d", expected, score)
	}
}

func TestScoreSkill_snykPenalty(t *testing.T) {
	s := Skill{Language: "go", SnykRisk: "Critical", ProjectType: "cli"}
	ctx := ProjectContext{Language: "go", ProjectType: "cli"}
	score := ScoreSkill(s, ctx)
	if score != 10 {
		t.Errorf("snyk penalty: expected 10 (40-30), got %d", score)
	}
}

func TestRecommendSkills_filtersBelowThreshold(t *testing.T) {
	skills := []Skill{
		{Name: "good", Language: "go", ProjectType: "cli", Publisher: "anthropics"},
		{Name: "bad", Language: "python", ProjectType: "web"},
	}
	ctx := ProjectContext{Language: "go", ProjectType: "cli"}
	results := RecommendSkills(skills, ctx, 50)
	if len(results) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(results))
	}
	if len(results) > 0 && results[0].Skill.Name != "good" {
		t.Errorf("expected 'good' skill, got %s", results[0].Skill.Name)
	}
}

func TestRecommendSkills_ordersByScore(t *testing.T) {
	skills := []Skill{
		{Name: "low", Language: "go", ProjectType: "cli"},
		{Name: "high", Language: "go", ProjectType: "cli", Publisher: "anthropics"},
	}
	ctx := ProjectContext{Language: "go", ProjectType: "cli"}
	results := RecommendSkills(skills, ctx, 0)
	if len(results) < 2 {
		t.Fatal("expected both skills")
	}
	if results[0].Score < results[1].Score {
		t.Error("expected results sorted by score descending")
	}
}
