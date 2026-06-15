package skilladvisor

// ProjectContext describes the project for scoring skills.
type ProjectContext struct {
	Language    string
	Framework   string
	ProjectType string // "cli", "web", "api", "mobile"
}

// Skill represents a discovered skill with audit info.
type Skill struct {
	Name            string
	Language        string
	Framework       string
	ProjectType     string
	Publisher       string
	Stars           int
	GenAgentTrustHub string // "Safe", "Unsafe", "Unknown"
	SocketAlerts    *int  // nil if unknown, 0 if no alerts
	SnykRisk        string // "Low", "Medium", "High", "Critical"
	SourceURL       string
}

// ScoreSkill returns a deterministic score for a skill given project context.
// Higher = more relevant. Threshold: ≥50 recommended, ≥75 strongly recommended.
func ScoreSkill(s Skill, ctx ProjectContext) int {
	score := 0

	// Language match: +10
	if s.Language != "" && s.Language == ctx.Language {
		score += 10
	}

	// Framework match: +20
	if s.Framework != "" && s.Framework == ctx.Framework {
		score += 20
	}

	// Project type match: +30
	if s.ProjectType != "" && s.ProjectType == ctx.ProjectType {
		score += 30
	}

	// Verified publishers: +50
	if isVerifiedPublisher(s.Publisher) {
		score += 50
	}

	// Gen Agent Trust Hub: +25 for Safe
	if s.GenAgentTrustHub == "Safe" {
		score += 25
	}

	// Socket 0 alerts: +15 (only if explicitly set to 0)
	if s.SocketAlerts != nil && *s.SocketAlerts == 0 {
		score += 15
	}

	// Snyk penalty for High/Critical: -30
	if s.SnykRisk == "High" || s.SnykRisk == "Critical" {
		score -= 30
	}

	return score
}

// verifiedPublishers is a curated list of trusted skill publishers.
var verifiedPublishers = map[string]bool{
	"anthropics":     true,
	"microsoft":      true,
	"nvidia":         true,
	"vercel-labs":    true,
	"vercel":         true,
	"google":         true,
	"openai":         true,
	"jeffallan":      true,
	"samber":         true,
	"affaan-m":       true,
	"mattpocock":     true,
	"gentleman":      true,
}

func isVerifiedPublisher(publisher string) bool {
	return verifiedPublishers[publisher]
}

// ScoredSkill pairs a skill with its recommendation score.
type ScoredSkill struct {
	Skill Skill
	Score int
}

// RecommendSkills filters and ranks skills for a project context.
// Returns skills with score ≥ threshold (default 50).
func RecommendSkills(skills []Skill, ctx ProjectContext, threshold int) []ScoredSkill {
	if threshold < 0 {
		threshold = 50
	}

	var results []ScoredSkill
	for _, s := range skills {
		score := ScoreSkill(s, ctx)
		if score >= threshold {
			results = append(results, ScoredSkill{Skill: s, Score: score})
		}
	}

	// Sort by score descending (simple bubble sort)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
