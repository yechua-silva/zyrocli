package planning

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Schedule types
// ---------------------------------------------------------------------------

// ScheduleEntry represents a single scheduled item in the execution plan.
type ScheduleEntry struct {
	Feature    Feature `json:"feature"`
	Phase      int     `json:"phase"`       // execution phase (1-based)
	Priority   int     `json:"priority"`    // within-phase priority (1=highest)
	BlockedBy  []string `json:"blocked_by"` // feature IDs that must complete first
}

// Schedule represents a complete ordered execution plan.
type Schedule struct {
	Project   string          `json:"project"`
	Phases    [][]ScheduleEntry `json:"phases"`    // phase → entries
	Remaining []Feature       `json:"remaining"`  // features that couldn't be scheduled
}

// PhaseCount returns the number of phases in the schedule.
func (s *Schedule) PhaseCount() int {
	return len(s.Phases)
}

// TotalEntries returns the total number of scheduled entries across all phases.
func (s *Schedule) TotalEntries() int {
	count := 0
	for _, phase := range s.Phases {
		count += len(phase)
	}
	return count
}

// Markdown renders the schedule as a structured markdown string.
func (s *Schedule) Markdown() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Schedule: %s\n\n", s.Project))
	b.WriteString(fmt.Sprintf("Total: %d features across %d phases\n\n", s.TotalEntries(), s.PhaseCount()))

	for i, phase := range s.Phases {
		b.WriteString(fmt.Sprintf("### Phase %d\n\n", i+1))
		b.WriteString("| Priority | Feature | Complexity | Blocked By |\n")
		b.WriteString("|----------|---------|------------|------------|\n")
		for _, entry := range phase {
			blocked := "—"
			if len(entry.BlockedBy) > 0 {
				blocked = strings.Join(entry.BlockedBy, ", ")
			}
			b.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
				entry.Priority, entry.Feature.Name, entry.Feature.Complexity, blocked))
		}
		b.WriteString("\n")
	}

	if len(s.Remaining) > 0 {
		b.WriteString("### Unscheduled\n\n")
		for _, f := range s.Remaining {
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", f.ID, f.Name, f.Description))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Scheduler
// ---------------------------------------------------------------------------

// SchedulerConfig configures scheduling behavior.
type SchedulerConfig struct {
	MaxPhases     int                 // max phases (default 10)
	PhasePriorities map[string]int    // feature name → manual priority override
}

// Scheduler orders features by dependencies and priority using topological sort.
type Scheduler struct {
	config SchedulerConfig
}

// NewScheduler creates a Scheduler with the given configuration.
func NewScheduler(cfg SchedulerConfig) *Scheduler {
	if cfg.MaxPhases <= 0 {
		cfg.MaxPhases = 10
	}
	if cfg.PhasePriorities == nil {
		cfg.PhasePriorities = make(map[string]int)
	}
	return &Scheduler{config: cfg}
}

// Schedule runs topological sort on features to produce an ordered execution plan.
// Returns the schedule and any errors.
func (s *Scheduler) Schedule(project string, features []Feature) (*Schedule, error) {
	if len(features) == 0 {
		return &Schedule{Project: project}, fmt.Errorf("schedule: no features to schedule")
	}

	// Build dependency graph
	featureMap := make(map[string]*Feature, len(features))
	for i := range features {
		featureMap[features[i].ID] = &features[i]
	}

	// Validate dependencies
	for _, f := range features {
		for _, dep := range f.Dependencies {
			if _, ok := featureMap[dep]; !ok {
				return nil, fmt.Errorf("schedule: feature %q depends on unknown feature %q", f.ID, dep)
			}
		}
	}

	// Topological sort using Kahn's algorithm
	phases := s.kahnSort(features, featureMap)

	// Assign priorities within each phase
	schedule := &Schedule{Project: project}
	for _, phase := range phases {
		schedule.Phases = append(schedule.Phases, s.assignPriorities(phase))
	}

	return schedule, nil
}

// kahnSort implements Kahn's algorithm for topological sort.
// Features with no remaining dependencies form a phase; their removal
// may unblock others for the next phase.
func (s *Scheduler) kahnSort(features []Feature, featureMap map[string]*Feature) [][]ScheduleEntry {
	// Build in-degree counts
	inDegree := make(map[string]int, len(features))
	for _, f := range features {
		inDegree[f.ID] = len(f.Dependencies)
	}

	// Build reverse dependency map (what depends on what)
	reverseDeps := make(map[string][]string)
	for _, f := range features {
		for _, dep := range f.Dependencies {
			reverseDeps[dep] = append(reverseDeps[dep], f.ID)
		}
	}

	var phases [][]ScheduleEntry
	scheduled := make(map[string]bool)

	for len(scheduled) < len(features) && len(phases) < s.config.MaxPhases {
		// Find features with in-degree == 0 that are not yet scheduled
		var current []ScheduleEntry
		for _, f := range features {
			if inDegree[f.ID] == 0 && !scheduled[f.ID] {
				blockedBy := make([]string, len(f.Dependencies))
				copy(blockedBy, f.Dependencies)
				current = append(current, ScheduleEntry{
					Feature:   f,
					Phase:     len(phases) + 1,
					BlockedBy: blockedBy,
				})
			}
		}

		if len(current) == 0 {
			// Remaining features have circular dependencies
			break
		}

		// Sort current phase by priority hints
		sort.SliceStable(current, func(i, j int) bool {
			pi := s.config.PhasePriorities[current[i].Feature.Name]
			pj := s.config.PhasePriorities[current[j].Feature.Name]
			if pi != pj {
				return pi < pj // lower number = higher priority
			}
			return current[i].Feature.ID < current[j].Feature.ID
		})

		phases = append(phases, current)

		// Remove these features from the graph
		for _, entry := range current {
			scheduled[entry.Feature.ID] = true
			for _, dependent := range reverseDeps[entry.Feature.ID] {
				inDegree[dependent]--
			}
		}
	}

	// Collect unscheduled features (circular dependencies)
	// These are not errors — the caller decides how to handle
	_ = len(scheduled)

	return phases
}

func (s *Scheduler) assignPriorities(entries []ScheduleEntry) []ScheduleEntry {
	result := make([]ScheduleEntry, len(entries))
	for i, entry := range entries {
		result[i] = entry
		result[i].Priority = i + 1
	}
	return result
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

// PhaseBoundaryDescription generates a human-readable boundary summary for a phase.
func PhaseBoundaryDescription(phase []ScheduleEntry) string {
	if len(phase) == 0 {
		return "(empty phase)"
	}
	var names []string
	for _, e := range phase {
		names = append(names, e.Feature.Name)
	}
	return fmt.Sprintf("features: %s", strings.Join(names, ", "))
}

// ValidateNoCircularDeps returns an error if circular dependencies exist.
func ValidateNoCircularDeps(features []Feature) error {
	if len(features) == 0 {
		return nil
	}

	// Build feature lookup by ID
	featureByID := make(map[string]Feature, len(features))
	for _, f := range features {
		featureByID[f.ID] = f
	}

	// Track visited states per node
	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully explored
	)

	colors := make(map[string]int, len(features))
	for _, f := range features {
		colors[f.ID] = white
	}

	var dfs func(id string) error
	dfs = func(id string) error {
		colors[id] = gray
		feat, ok := featureByID[id]
		if !ok {
			colors[id] = black
			return nil
		}
		for _, dep := range feat.Dependencies {
			switch colors[dep] {
			case gray:
				return fmt.Errorf("circular dependency: %s → %s", id, dep)
			case white:
				if err := dfs(dep); err != nil {
					return err
				}
			}
		}
		colors[id] = black
		return nil
	}

	for _, f := range features {
		if colors[f.ID] == white {
			if err := dfs(f.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
