package helix

import (
	"context"
	"fmt"
)

// IndexType enum para tipos de índice
type IndexType string

const (
	IndexVector   IndexType = "vector"
	IndexText     IndexType = "text"
	IndexEquality IndexType = "equality"
	IndexRange    IndexType = "range"
	IndexUnique   IndexType = "unique"
)

// IndexSpec describe un índice a crear
type IndexSpec struct {
	Label      string
	Property   string
	IndexType  IndexType
	Dimensions int // solo para vector indexes
}

// EnsureIndexes crea índices si no existen
func EnsureIndexes(ctx context.Context, client *Client, specs []IndexSpec) error {
	for _, spec := range specs {
		if err := client.Exec(ctx, AddIndex(spec.Label, spec.Property, string(spec.IndexType)), nil); err != nil {
			return fmt.Errorf("index %s.%s: %w", spec.Label, spec.Property, err)
		}
	}
	return nil
}

// DefaultIndexes retorna los índices recomendados para el schema de Zyro
func DefaultIndexes() []IndexSpec {
	return []IndexSpec{
		{Label: "Project", Property: "name", IndexType: IndexEquality},
		{Label: "Project", Property: "status", IndexType: IndexEquality},
		{Label: "Task", Property: "name", IndexType: IndexEquality},
		{Label: "Task", Property: "phase", IndexType: IndexEquality},
		{Label: "Task", Property: "status", IndexType: IndexEquality},
		{Label: "CodeNode", Property: "path", IndexType: IndexEquality},
		{Label: "CodeNode", Property: "language", IndexType: IndexEquality},
		{Label: "Skill", Property: "name", IndexType: IndexEquality},
		{Label: "Skill", Property: "type", IndexType: IndexEquality},
		{Label: "Pattern", Property: "name", IndexType: IndexEquality},
		{Label: "Fact", Property: "fact_type", IndexType: IndexEquality},
		{Label: "Fact", Property: "phase", IndexType: IndexEquality},
		{Label: "Fact", Property: "is_active", IndexType: IndexEquality},
		{Label: "Fact", Property: "content", IndexType: IndexText},
		{Label: "Fact", Property: "embedding", IndexType: IndexVector, Dimensions: 1536},
		{Label: "Project", Property: "description", IndexType: IndexText},
		{Label: "CodeNode", Property: "summary", IndexType: IndexText},
		{Label: "Skill", Property: "description", IndexType: IndexText},
	}
}
