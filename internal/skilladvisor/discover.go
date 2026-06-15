package skilladvisor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// skillsShResult represents a single skill from the skills.sh API.
type skillsShResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Source      string `json:"source"`
	Publisher   string `json:"publisher"`
	Language    string `json:"language"`
	Framework   string `json:"framework"`
	ProjectType string `json:"project_type"`
}

// DiscoverFromSkillsSh searches skills.sh for skills matching the query.
// Returns up to limit results.
func DiscoverFromSkillsSh(query string, limit int) ([]Skill, error) {
	if limit <= 0 {
		limit = 10
	}

	// Use the skills.sh search API
	apiURL := fmt.Sprintf("https://skills.sh/api/search?q=%s&limit=%d", url.QueryEscape(query), limit)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh: HTTP %d", resp.StatusCode)
	}

	var results []skillsShResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("skills.sh: decode: %w", err)
	}

	skills := make([]Skill, 0, len(results))
	for _, r := range results {
		skills = append(skills, Skill{
			Name:      r.Name,
			Publisher: r.Publisher,
			Stars:     r.Stars,
			Language:  r.Language,
			SourceURL: r.Source,
		})
	}

	return skills, nil
}
