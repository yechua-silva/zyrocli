package tui

// ── Embedding model options ──────────────────────────────────────────────

var EmbeddingOptions = []SelectOption{
	{
		Key:         "mxbai-embed-large:latest",
		Label:       "mxbai-embed-large",
		Detail:      "1024 dims · ~0.5B",
		Description: "Recomendado — balance calidad/rendimiento",
	},
	{
		Key:         "nomic-embed-text:latest",
		Label:       "nomic-embed-text",
		Detail:      "768 dims · ~0.1B",
		Description: "Más rápido en CPU, bueno para equipos modestos",
	},
	{
		Key:         "all-minilm:latest",
		Label:       "all-minilm",
		Detail:      "384 dims · ~0.02B",
		Description: "Ultraligero, ideal para CPU o Raspberry Pi",
	},
}

// ── Chat model options ───────────────────────────────────────────────────

var ChatOptions = []SelectOption{
	{
		Key:         "llama3.2:3b",
		Label:       "llama3.2:3b",
		Detail:      "~2.0 GB · 3B params",
		Description: "Recomendado — Meta Llama 3.2, balance velocidad/calidad",
	},
	{
		Key:         "phi4-mini:3.8b",
		Label:       "phi4-mini:3.8b",
		Detail:      "~2.5 GB · 3.8B params",
		Description: "Microsoft Phi-4 Mini, excelente para JSON y structured output",
	},
	{
		Key:         "qwen3.5:0.5b",
		Label:       "qwen3.5:0.5b",
		Detail:      "~0.4 GB · 0.5B params",
		Description: "Ultraligero, funciona en cualquier parte",
	},
	{
		Key:         "gemma3:2b",
		Label:       "gemma3:2b",
		Detail:      "~1.5 GB · 2B params",
		Description: "Google Gemma 3, buena relación calidad/tamaño",
	},
	{
		Key:         "mistral:7b",
		Label:       "mistral:7b",
		Detail:      "~4.1 GB · 7B params",
		Description: "Más preciso pero pesado — requiere GPU o mucha RAM",
	},
}

// ── ModelsFlow ───────────────────────────────────────────────────────────

// RunModelsFlow ejecuta el flujo completo de configuración de modelos.
// Retorna la config seleccionada o nil si cancela.
func RunModelsFlow() map[string]string {
	// Paso 1: Elegir modelo de embeddings
	embedModel := RunSelect(
		"Modelo de Embeddings",
		"Selecciona el modelo para generar vectores (búsqueda semántica, memoria):",
		EmbeddingOptions,
	)
	if embedModel == "" {
		return nil
	}

	// Paso 2: Elegir modelo chat
	chatModel := RunSelect(
		"Modelo Chat (LLM)",
		"Selecciona el modelo para procesar texto y extraer hechos:",
		ChatOptions,
	)
	if chatModel == "" {
		return nil
	}

	return map[string]string{
		"embedding_model": embedModel,
		"chat_model":      chatModel,
	}
}
