// Package tokens estima el conteo de tokens de un texto.
// Fórmula: len(text)/4 (estándar OpenAI: 1 token ≈ 4 chars para inglés/código)
// Fuente: https://platform.openai.com/tokenizer
//
// Exactitud: ±20% para rangos típicos (1k-100k chars).
// Límites: no soporta chino/japonés/coreano (2-5 chars/token).
// Para mediciones precisas, usar tiktoken (https://github.com/openai/tiktoken).
package tokens

// Count estima tokens de un texto.
// Usa ceiling division para no subestimar textos cortos.
func Count(text string) int64 {
	n := len(text)
	if n == 0 {
		return 0
	}
	return int64((n + 3) / 4)
}

// Breakdown desglose detallado del conteo.
type Breakdown struct {
	Total      int64  `json:"total"`
	Characters int    `json:"characters"`
	Method     string `json:"method"` // siempre "char_div_4"
}

// CountBreakdown retorna un desglose detallado del conteo.
func CountBreakdown(text string) Breakdown {
	return Breakdown{
		Total:      Count(text),
		Characters: len(text),
		Method:     "char_div_4",
	}
}
