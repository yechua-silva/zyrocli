package tui

import (
	"strings"
	"testing"
)

func TestRenderBrand(t *testing.T) {
	result := RenderBrand()
	if result == "" {
		t.Error("RenderBrand() should not be empty")
	}
	if !strings.Contains(result, "█") {
		t.Error("RenderBrand() should contain block chars")
	}
}

func TestRenderLogo(t *testing.T) {
	result := RenderLogo()
	if result == "" {
		t.Error("RenderLogo() should not be empty")
	}
}

func TestRenderWelcome(t *testing.T) {
	result := RenderWelcome("Test subtitle")
	if result == "" {
		t.Error("RenderWelcome() should not be empty")
	}
	if !strings.Contains(result, "Test subtitle") {
		t.Error("RenderWelcome() should contain subtitle")
	}
}

func TestRenderFullBanner(t *testing.T) {
	result := RenderFullBanner("Test subtitle")
	if result == "" {
		t.Error("RenderFullBanner() should not be empty")
	}
}

func TestSuccess(t *testing.T) {
	result := Success("test message")
	if !strings.Contains(result, "✓") {
		t.Error("Success() should contain checkmark")
	}
	if !strings.Contains(result, "test message") {
		t.Error("Success() should contain the message")
	}
}

func TestWarning(t *testing.T) {
	result := Warning("test warning")
	if !strings.Contains(result, "⚠") {
		t.Error("Warning() should contain warning symbol")
	}
}

func TestErrorStr(t *testing.T) {
	result := ErrorStr("test error")
	if !strings.Contains(result, "✗") {
		t.Error("ErrorStr() should contain X mark")
	}
}

func TestInfo(t *testing.T) {
	result := Info("test info")
	if !strings.Contains(result, "•") {
		t.Error("Info() should contain bullet")
	}
}

func TestRenderNewLogo(t *testing.T) {
	result := RenderNewLogo()
	if result == "" {
		t.Error("RenderNewLogo() returned empty")
	}
	if !strings.Contains(result, "@") && !strings.Contains(result, "#") {
		t.Error("RenderNewLogo() missing expected ASCII characters")
	}
}

func TestMaxLineWidth(t *testing.T) {
	art := "hello\nworld!"
	width := maxLineWidth(art)
	if width != 6 {
		t.Errorf("maxLineWidth() = %d, want 6", width)
	}
}
