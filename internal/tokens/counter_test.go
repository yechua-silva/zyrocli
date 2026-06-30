package tokens

import (
	"testing"
)

func TestCount_Empty(t *testing.T) {
	if got := Count(""); got != 0 {
		t.Errorf("Count('') = %d, want 0", got)
	}
}

func TestCount_Short(t *testing.T) {
	got := Count("abc")
	if got != 1 {
		t.Errorf("Count('abc') = %d, want 1 (4 chars \u2192 1 token)", got)
	}
}

func TestCount_Exact(t *testing.T) {
	got := Count("abcdefgh") // 8 chars
	if got != 2 {
		t.Errorf("Count('abcdefgh') = %d, want 2", got)
	}
}

func TestCount_Long(t *testing.T) {
	// 1000 chars \u2192 ~250 tokens
	text := ""
	for i := 0; i < 100; i++ {
		text += "hello world " // 12 chars each
	}
	got := Count(text)
	expected := int64((len(text) + 3) / 4)
	if got != expected {
		t.Errorf("Count(long) = %d, want %d", got, expected)
	}
}

func TestCount_Unicode(t *testing.T) {
	// Caracteres UTF-8 multi-byte
	text := "café français 中文 español русский"
	got := Count(text)
	if got == 0 {
		t.Error("Count(unicode) = 0, want > 0")
	}
}

func TestCountBreakdown(t *testing.T) {
	b := CountBreakdown("hello world")
	if b.Total != 3 {
		t.Errorf("Breakdown.Total = %d, want 3", b.Total)
	}
	if b.Characters != 11 {
		t.Errorf("Breakdown.Characters = %d, want 11", b.Characters)
	}
	if b.Method != "char_div_4" {
		t.Errorf("Breakdown.Method = %s, want char_div_4", b.Method)
	}
}

func TestCount_ExactFour(t *testing.T) {
	got := Count("abcd") // 4 chars → 1 token (exacto)
	if got != 1 {
		t.Errorf("Count('abcd') = %d, want 1", got)
	}
}
