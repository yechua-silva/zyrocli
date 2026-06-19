package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/secko/zyrocli/internal/memory"
)

// MemoryHooks conecta la memoria causal con el scheduler.
// Inyecta contexto relevante antes de ejecutar una fase (PrePhase)
// y extrae/guarda hechos después de ejecutar una fase (PostPhase).
type MemoryHooks struct {
	store             memory.EngramStore
	factExtractorPath string
}

// NewMemoryHooks crea hooks de memoria causal.
// store: implementación de EngramStore (p.ej. HelixEngramStore).
// factExtractorPath: ruta al script Python fact_extractor.py.
func NewMemoryHooks(store memory.EngramStore, factExtractorPath string) *MemoryHooks {
	return &MemoryHooks{
		store:             store,
		factExtractorPath: factExtractorPath,
	}
}

// PrePhase recupera memorias relevantes y las formatea como contexto
// inyectable en el prompt del agente antes de ejecutar una fase.
// Retorna string vacío si no hay memorias o si el store es nil.
func (h *MemoryHooks) PrePhase(ctx context.Context, phase Phase, taskDesc string) (string, error) {
	if h.store == nil {
		return "", nil
	}

	results, err := h.store.RecallMemories(ctx, memory.RecallOpts{
		QueryText:   taskDesc,
		MaxResults:  10,
		MinSalience: 0.2,
		Phase:       string(phase),
	})
	if err != nil {
		return "", fmt.Errorf("memory pre-phase: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	return formatMemoryForPrompt(results), nil
}

// PostPhase extrae hechos del log de conversación usando fact_extractor.py
// y los persiste en la memoria causal.
func (h *MemoryHooks) PostPhase(ctx context.Context, phase Phase, conversationLog string) error {
	if h.store == nil || conversationLog == "" {
		return nil
	}

	// Si hay extractor Python configurado, usarlo
	if h.factExtractorPath != "" {
		return h.runFactExtractor(ctx, conversationLog, string(phase))
	}

	// Fallback: extracción simple integrada
	log.Printf("[memory] PostPhase: usando extractor simple (factExtractorPath vacío)")
	return h.extractSimpleFacts(ctx, conversationLog, string(phase))
}

// runFactExtractor ejecuta el script Python fact_extractor.py
// para extracción avanzada de hechos usando LLM o patrones.
func (h *MemoryHooks) runFactExtractor(ctx context.Context, logText string, phase string) error {
	// Construir JSON de entrada
	input := map[string]string{
		"conversation": logText,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("memory: marshal input: %w", err)
	}

	// Ejecutar fact_extractor.py con el log
	cmd := exec.CommandContext(ctx, "python3", h.factExtractorPath,
		"--input", "-",
		"--phase", phase,
	)
	cmd.Stdin = strings.NewReader(string(inputJSON))
	output, err := cmd.Output()
	if err != nil {
		// Soft fail: no bloquear la fase por un error de extracción
		log.Printf("[memory] fact_extractor error: %v (stderr: %s)", err, string(output))
		return nil
	}

	// Parsear resultado
	var result struct {
		Facts []memory.Fact `json:"facts"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("[memory] fact_extractor parse error: %v", err)
		return nil
	}

	// Guardar cada fact
	for i := range result.Facts {
		f := &result.Facts[i]
		f.Source = "extractor:postphase"
		f.Phase = phase
		if f.Salience == 0 {
			f.Salience = 0.5
		}
		if f.Confidence == 0 {
			f.Confidence = 0.6
		}
		if f.CreatedAt.IsZero() {
			f.CreatedAt = time.Now()
		}
		f.IsActive = true

		if _, err := h.store.SaveFact(ctx, f); err != nil {
			log.Printf("[memory] error saving fact: %v", err)
		}
	}

	log.Printf("[memory] PostPhase: %d facts saved from phase %s", len(result.Facts), phase)
	return nil
}

// extractSimpleFacts extrae hechos básicos sin depender del script Python.
// Usa palabras clave simples como fallback.
func (h *MemoryHooks) extractSimpleFacts(ctx context.Context, logText string, phase string) error {
	logLower := strings.ToLower(logText)
	keywords := map[string]string{
		// Spanish
		"decidimos":    "decision",
		"elegimos":     "decision",
		"prefiero":     "preference",
		"observo":      "observation",
		"noto":         "observation",
		"dependemos":   "dependency",
		"requiere":     "dependency",
		"acordamos":    "decision",
		"cambiamos":    "change",
		"bloquea":      "blocker",
		"arquitectura": "architecture",
		"rendimiento":  "performance",
		"seguridad":    "security",
		"deprecado":    "deprecation",
		// English
		"decision":     "decision",
		"error":        "error",
		"bug":          "error",
		"pattern":      "pattern",
		"dependency":   "dependency",
		"observation":  "observation",
		"fix":          "fix",
		"implement":    "implementation",
		"refactor":     "refactoring",
		"deprecated":   "deprecation",
		"breaking":     "breaking_change",
		"architecture": "architecture",
		"performance":  "performance",
		"security":     "security",
		"todo":         "todo",
		"note":         "note",
		"warning":      "warning",
		"critical":     "critical",
		"resolved":     "resolved",
		"blocked":      "blocker",
	}

	hitCount := 0
	for keyword, factType := range keywords {
		if strings.Contains(logLower, keyword) {
			hitCount++
			// Extraer contexto alrededor de la palabra clave
			idx := strings.Index(logLower, keyword)
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + len(keyword) + 100
			if end > len(logText) {
				end = len(logText)
			}
			content := strings.TrimSpace(logText[start:end])

			fact := &memory.Fact{
				Type:       memory.FactType(factType),
				Content:    content,
				Salience:   0.5,
				Confidence: 0.6,
				Source:     "memory:simple-fallback",
				Phase:      phase,
				IsActive:   true,
				DecayRate:  0.05,
				CreatedAt:  time.Now(),
			}

			if _, err := h.store.SaveFact(ctx, fact); err != nil {
				log.Printf("[memory] error saving simple fact: %v", err)
			}
		}
	}

	if hitCount > 0 {
		log.Printf("[memory] PostPhase simple: %d keyword(s) found, facts saved for phase %s", hitCount, phase)
	} else {
		log.Printf("[memory] PostPhase simple: no keywords found in phase %s log", phase)
	}

	return nil
}

// formatMemoryForPrompt formatea una lista de MemoryResult como bloque
// de contexto inyectable en el prompt del agente.
func formatMemoryForPrompt(results []*memory.MemoryResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## MEMORIA CAUSAL (hechos previos relevantes)\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] ", i+1, r.Fact.Type))
		sb.WriteString(r.Fact.Content)
		sb.WriteString(fmt.Sprintf(" (confianza: %.0f%%, fase: %s)\n",
			r.Fact.Confidence*100, r.Fact.Phase))
	}

	return sb.String()
}
