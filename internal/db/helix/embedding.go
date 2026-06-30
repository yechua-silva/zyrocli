package helix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/secko/zyrocli/internal/setup"
)

// EmbeddingProvider define el origen de embeddings
type EmbeddingProvider string

const (
	ProviderOpenAI EmbeddingProvider = "openai"
	ProviderOllama EmbeddingProvider = "ollama"
)

// EmbeddingConfig configura el servicio de embeddings
type EmbeddingConfig struct {
	Provider   EmbeddingProvider
	Model      string
	Dims       int
	APIKey     string
	BaseURL    string
	BatchSize  int
	CacheSize  int
	MaxRetries int
	Timeout    time.Duration
}

// DefaultEmbeddingConfig retorna configuración por defecto
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		Provider:   ProviderOpenAI,
		Model:      "text-embedding-3-small",
		Dims:       1536,
		BatchSize:  20,
		CacheSize:  1000,
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
}

// EmbeddingService genera embeddings para texto
type EmbeddingService struct {
	config EmbeddingConfig
	client *http.Client
	cache  sync.Map   // map[string][]float32
	keys   []string   // FIFO order for eviction
	mu     sync.Mutex // guards keys slice
}

// NewEmbeddingService crea un nuevo servicio de embeddings
func NewEmbeddingService(config EmbeddingConfig) *EmbeddingService {
	return &EmbeddingService{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// evictIfNeeded elimina entradas más antiguas si el caché excede CacheSize (FIFO)
func (s *EmbeddingService) evictIfNeeded() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.CacheSize <= 0 || len(s.keys) <= s.config.CacheSize {
		return
	}

	// Eliminar el 10% más antiguo cuando se excede el límite
	evictCount := s.config.CacheSize / 10
	if evictCount < 1 {
		evictCount = 1
	}

	for i := 0; i < evictCount && i < len(s.keys); i++ {
		s.cache.Delete(s.keys[i])
	}
	s.keys = s.keys[evictCount:]
}

// Embed genera un embedding para un texto
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embedding: empty text")
	}

	// Check cache
	if cached, ok := s.cache.Load(text); ok {
		return cached.([]float32), nil
	}

	// Generar embedding
	result, err := s.embedWithRetry(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("embedding: no result")
	}

	// Store cache
	s.cache.Store(text, result[0])
	s.mu.Lock()
	s.keys = append(s.keys, text)
	s.mu.Unlock()
	s.evictIfNeeded()

	return result[0], nil
}

// EmbedBatch genera embeddings para múltiples textos
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var results [][]float32
	var uncached []string
	uncachedIdx := make([]int, 0)

	for i, text := range texts {
		if cached, ok := s.cache.Load(text); ok {
			results = append(results, cached.([]float32))
		} else {
			results = append(results, nil) // placeholder
			uncached = append(uncached, text)
			uncachedIdx = append(uncachedIdx, i)
		}
	}

	if len(uncached) == 0 {
		return results, nil
	}

	// Procesar en batches
	for i := 0; i < len(uncached); i += s.config.BatchSize {
		end := i + s.config.BatchSize
		if end > len(uncached) {
			end = len(uncached)
		}
		batch := uncached[i:end]

		embeddings, err := s.embedWithRetry(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embed batch %d: %w", i/s.config.BatchSize, err)
		}

		for j, emb := range embeddings {
			idx := uncachedIdx[i+j]
			results[idx] = emb
			s.cache.Store(uncached[i+j], emb)
			s.mu.Lock()
			s.keys = append(s.keys, uncached[i+j])
			s.mu.Unlock()
		}
	}

	s.evictIfNeeded()

	return results, nil
}

func (s *EmbeddingService) embedWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(100*attempt) * time.Millisecond)
		}

		var embeddings [][]float32
		var err error

		switch s.config.Provider {
		case ProviderOpenAI:
			embeddings, err = s.embedOpenAI(ctx, texts)
		case ProviderOllama:
			embeddings, err = s.embedOllama(ctx, texts)
		default:
			return nil, fmt.Errorf("unknown provider: %s", s.config.Provider)
		}

		if err == nil {
			return embeddings, nil
		}
		lastErr = err

		// Fallback: si OpenAI falla, intentar con Ollama
		if s.config.Provider == ProviderOpenAI {
			ollamaEmb, ollamaErr := s.embedOllama(ctx, texts)
			if ollamaErr == nil {
				return ollamaEmb, nil
			}
		}
	}

	return nil, fmt.Errorf("embedding after %d retries: %w", s.config.MaxRetries, lastErr)
}

func (s *EmbeddingService) embedOpenAI(ctx context.Context, texts []string) ([][]float32, error) {
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	reqBody := map[string]interface{}{
		"input": texts,
		"model": s.config.Model,
	}
	if s.config.Dims > 0 {
		reqBody["dimensions"] = s.config.Dims
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai decode: %w", err)
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		emb := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			emb[j] = float32(v)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

func (s *EmbeddingService) embedOllama(ctx context.Context, texts []string) ([][]float32, error) {
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = setup.GetOllamaURL()
	}
	model := s.config.Model
	if model == "" {
		model = setup.GetEmbeddingModel()
	}

	var embeddings [][]float32
	for _, text := range texts {
		reqBody := map[string]interface{}{
			"model":  model,
			"prompt": text,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/embeddings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ollama request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("ollama decode: %w", err)
		}

		emb := make([]float32, len(result.Embedding))
		for i, v := range result.Embedding {
			emb[i] = float32(v)
		}
		embeddings = append(embeddings, emb)
	}

	return embeddings, nil
}
