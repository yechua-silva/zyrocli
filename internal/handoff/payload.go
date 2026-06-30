package handoff

// Source identifies the origin of the handoff contract.
type Source struct {
	System string `yaml:"system"`           // required
	URL    string `yaml:"url,omitempty"`
}

// Project describes the software project this handoff targets.
type Project struct {
	Name       string `yaml:"name"`           // required
	Language   string `yaml:"language"`       // required
	Repository string `yaml:"repository,omitempty"`
}

// ValidatedIdea captures the validated problem and solution.
type ValidatedIdea struct {
	Problem   string `yaml:"problem,omitempty"`
	Solution  string `yaml:"solution,omitempty"`
	Rationale string `yaml:"rationale,omitempty"`
}

// AcceptanceCriteriaInfo represents an acceptance criterion for structured handoff tracking.
type AcceptanceCriteriaInfo struct {
	ID          string `yaml:"id" json:"id"`
	Description string `yaml:"description" json:"description"`
	Status      string `yaml:"status" json:"status"`
}

// AcceptanceSummary resumen global de acceptance criteria al final del pipeline.
type AcceptanceSummary struct {
	Total    int `yaml:"total" json:"total"`
	Verified int `yaml:"verified" json:"verified"`
	Failed   int `yaml:"failed" json:"failed"`
	Pending  int `yaml:"pending" json:"pending"`
}

// UserStory captures a single user story for the change.
type UserStory struct {
	Story      string                   `yaml:"story,omitempty"`
	Acceptance string                   `yaml:"acceptance,omitempty"` // se mantiene por backward compat
	Criteria   []AcceptanceCriteriaInfo `yaml:"criteria,omitempty"`   // structured criteria (NUEVO)
}

// MVP defines the minimum viable product scope.
type MVP struct {
	Scope    string   `yaml:"scope,omitempty"`
	Features []string `yaml:"features,omitempty"`
}

// ApprovalPoint defines a governance approval gate.
type ApprovalPoint struct {
	Name       string `yaml:"name,omitempty"`
	ApprovedBy string `yaml:"approved_by,omitempty"`
}

// Governance defines project governance and conventions.
type Governance struct {
	Mode      string          `yaml:"mode"`       // required
	Module    string          `yaml:"module,omitempty"`
	GoVersion string          `yaml:"go_version,omitempty"`
	StrictTDD bool            `yaml:"strict_tdd,omitempty"`
	Approvals []ApprovalPoint `yaml:"approvals,omitempty"`
}

// Testing defines the testing strategy.
type Testing struct {
	Strategy string `yaml:"strategy"`    // required
	Golden   bool   `yaml:"golden,omitempty"`
	Mock     string `yaml:"mock,omitempty"`
}

// Limits defines change constraints.
type Limits struct {
	MaxTasks     int    `yaml:"max_tasks,omitempty"`
	MaxLines     int    `yaml:"max_lines,omitempty"`
	MaxLoops     int    `yaml:"max_loops,omitempty"`
	PhaseTimeout string `yaml:"phase_timeout,omitempty"`
	ChainedPRs   bool   `yaml:"chained_prs,omitempty"`
}

// Payload is the top-level handoff structure read from handoff.yaml.
type Payload struct {
	Version          string             `yaml:"version"`
	Source           Source             `yaml:"source"`
	Project          Project            `yaml:"project"`
	ValidatedIdea    ValidatedIdea      `yaml:"validated_idea,omitempty"`
	UserStory        UserStory          `yaml:"user_story,omitempty"`
	MVP              MVP                `yaml:"mvp,omitempty"`
	Governance       Governance         `yaml:"governance"`
	Testing          Testing            `yaml:"testing"`
	Limits           Limits             `yaml:"limits,omitempty"`
	Capabilities     []string           `yaml:"capabilities,omitempty"`  // project capabilities for C-I-O traceability
	Dependencies     []string           `yaml:"dependencies,omitempty"`  // external dependencies for C-I-O traceability
	AcceptanceStatus *AcceptanceSummary `yaml:"acceptance_status,omitempty"` // NUEVO: resumen global de criteria
}
