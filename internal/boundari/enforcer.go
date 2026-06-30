package boundari

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Enforcer evalua políticas Boundari
type Enforcer struct {
	policy  *Policy
	usage   BudgetUsage
	started bool
}

// NewEnforcer crea un Enforcer con una política
func NewEnforcer(policy *Policy) *Enforcer {
	return &Enforcer{
		policy:  policy,
		usage:   BudgetUsage{StartedAt: time.Now()},
		started: true,
	}
}

// CheckTool verifica si una tool está permitida según la política
func (e *Enforcer) CheckTool(toolName string, args map[string]any) EnforcementResult {
	if !e.started {
		e.usage.StartedAt = time.Now()
		e.started = true
	}

	// Budget checks
	if e.usage.ToolCalls >= e.policy.Budget.MaxToolCalls {
		return EnforcementResult{
			Allowed: false, Tool: toolName, Phase: e.policy.Phase,
			Reason: fmt.Sprintf("budget exceeded: max %d tool calls", e.policy.Budget.MaxToolCalls),
		}
	}
	if time.Since(e.usage.StartedAt) > time.Duration(e.policy.Budget.MaxRuntimeSecs)*time.Second {
		return EnforcementResult{
			Allowed: false, Tool: toolName, Phase: e.policy.Phase,
			Reason: fmt.Sprintf("budget exceeded: max %d seconds", e.policy.Budget.MaxRuntimeSecs),
		}
	}

	// Find tool rule
	for _, rule := range e.policy.Tools {
		if rule.Name == toolName {
			switch rule.Action {
			case ActionDeny:
				return EnforcementResult{
					Allowed: false, Tool: toolName, Phase: e.policy.Phase,
					Reason: fmt.Sprintf("denied by policy: %s", rule.Name),
				}
			case ActionAllow, ActionApproval:
				result := EnforcementResult{
					Allowed: true, Tool: toolName, Phase: e.policy.Phase,
					Reason: "allowed by policy",
				}
				if rule.RequireApproval || rule.Action == ActionApproval {
					result.Allowed = false
					result.Reason = "requires approval"
					return result
				}
				e.usage.ToolCalls++
				return result
			}
		}
	}

	// Tool not in policy → deny by default
	return EnforcementResult{
		Allowed: false, Tool: toolName, Phase: e.policy.Phase,
		Reason: fmt.Sprintf("tool %s not in policy", toolName),
	}
}

// IsBudgetExceeded verifica si se excedió el presupuesto
func (e *Enforcer) IsBudgetExceeded() bool {
	if e.usage.ToolCalls >= e.policy.Budget.MaxToolCalls {
		return true
	}
	if time.Since(e.usage.StartedAt) > time.Duration(e.policy.Budget.MaxRuntimeSecs)*time.Second {
		return true
	}
	return false
}

// Usage devuelve el uso actual del presupuesto
func (e *Enforcer) Usage() BudgetUsage {
	return e.usage
}

// Reset reinicia el uso del presupuesto
func (e *Enforcer) Reset() {
	e.usage = BudgetUsage{}
	e.started = false
}

// auditLogger registra eventos de auditoría
type auditLogger struct {
	events []AuditEvent
}

var defaultAuditLogger = &auditLogger{}

// LogAudit registra un evento
func LogAudit(event AuditEvent) {
	event.Timestamp = time.Now()
	defaultAuditLogger.events = append(defaultAuditLogger.events, event)
}

// SaveAuditLog escribe el log de auditoría en formato JSONL
func SaveAuditLog(path string) error {
	var lines []string
	for _, e := range defaultAuditLogger.events {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("audit marshal: %w", err)
		}
		lines = append(lines, string(data))
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("audit mkdir: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ClearAuditLog limpia todos los eventos
func ClearAuditLog() {
	defaultAuditLogger.events = nil
}

// GetAuditEvents devuelve los eventos registrados
func GetAuditEvents() []AuditEvent {
	return defaultAuditLogger.events
}
