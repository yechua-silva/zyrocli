package tui

import (
	"net/http"
	"time"

	"github.com/yechua-silva/zyrocli/internal/setup"
)

// ServiceStatus represents the status of a service.
type ServiceStatus struct {
	Name    string
	Running bool
	URL     string
}

// CheckHelixDB verifica que HelixDB esté corriendo.
func CheckHelixDB() ServiceStatus {
	url := setup.GetHelixDBURL()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return ServiceStatus{
			Name:    "HelixDB",
			Running: false,
			URL:     url,
		}
	}
	defer resp.Body.Close()
	return ServiceStatus{
		Name:    "HelixDB",
		Running: resp.StatusCode == 200,
		URL:     url,
	}
}

// CheckOllama verifica que Ollama esté corriendo.
func CheckOllama() ServiceStatus {
	url := setup.GetOllamaURL()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/api/tags")
	if err != nil {
		return ServiceStatus{
			Name:    "Ollama",
			Running: false,
			URL:     url,
		}
	}
	defer resp.Body.Close()
	return ServiceStatus{
		Name:    "Ollama",
		Running: resp.StatusCode == 200,
		URL:     url,
	}
}

// FormatServiceStatus returns a colored string for service status.
func FormatServiceStatus(s ServiceStatus) string {
	if s.Running {
		return Success(s.Name + " ✅ corriendo en " + s.URL)
	}
	return ErrorStr(s.Name + " ❌ no responde en " + s.URL)
}
