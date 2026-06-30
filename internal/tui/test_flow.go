package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/secko/zyrocli/internal/setup"
)

// TestResult holds the result of a model test.
type TestResult struct {
	Name   string
	OK     bool
	Detail string
}

// TestEmbedding prueba el modelo de embeddings contra Ollama.
func TestEmbedding(model string, timeout int) TestResult {
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	jsonBody, err := json.Marshal(map[string]string{
		"model":  model,
		"prompt": "ZyroAgentCLI test de embedding — verificación de instalación.",
	})
	if err != nil {
		return TestResult{
			Name:   "Embeddings",
			OK:     false,
			Detail: fmt.Sprintf("Error creando JSON: %v", err),
		}
	}

	start := time.Now()
	resp, err := client.Post(
		setup.GetOllamaURL()+"/api/embeddings",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return TestResult{
			Name:   "Embeddings",
			OK:     false,
			Detail: fmt.Sprintf("Error conectando a Ollama: %v", err),
		}
	}
	defer resp.Body.Close()

	duration := time.Since(start).Round(time.Millisecond)

	if resp.StatusCode != 200 {
		return TestResult{
			Name:   "Embeddings",
			OK:     false,
			Detail: fmt.Sprintf("HTTP %d (%v)", resp.StatusCode, duration),
		}
	}

	return TestResult{
		Name:   "Embeddings",
		OK:     true,
		Detail: fmt.Sprintf("Respondió en %v", duration),
	}
}

// TestChat prueba el modelo chat contra Ollama.
func TestChat(model string, timeout int) TestResult {
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	jsonBody, err := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": "Responde solo OK si funciono.",
		"stream": false,
	})
	if err != nil {
		return TestResult{
			Name:   "Chat",
			OK:     false,
			Detail: fmt.Sprintf("Error creando JSON: %v", err),
		}
	}

	start := time.Now()
	resp, err := client.Post(
		setup.GetOllamaURL()+"/api/generate",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return TestResult{
			Name:   "Chat",
			OK:     false,
			Detail: fmt.Sprintf("Error conectando a Ollama: %v", err),
		}
	}
	defer resp.Body.Close()

	duration := time.Since(start).Round(time.Millisecond)

	if resp.StatusCode != 200 {
		return TestResult{
			Name:   "Chat",
			OK:     false,
			Detail: fmt.Sprintf("HTTP %d (%v)", resp.StatusCode, duration),
		}
	}

	return TestResult{
		Name:   "Chat",
		OK:     true,
		Detail: fmt.Sprintf("Respondió en %v", duration),
	}
}

// FormatTestResult returns a colored string for a test result.
func FormatTestResult(r TestResult) string {
	if r.OK {
		return Success(fmt.Sprintf("%s: %s", r.Name, r.Detail))
	}
	return ErrorStr(fmt.Sprintf("%s: %s", r.Name, r.Detail))
}
