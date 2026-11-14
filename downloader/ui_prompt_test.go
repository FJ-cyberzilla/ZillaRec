package downloader

import (
	"strings"
	"testing"

	"zillarec/config"
)

func TestParsePercent(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"45.5%", 45.5},
		{"100%", 100},
		{"0%", 0},
		{"N/A", 0},
		{"", 0},
		{" 78.2 % ", 78.2},
	}

	for _, tt := range tests {
		got := ParsePercent(tt.input)
		if got != tt.expected {
			t.Errorf("ParsePercent(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}

func TestPromptQualityDefault(t *testing.T) {
	r := NewRenderer(ioDiscard{})
	profiles := config.QualityProfiles()

	// Simulating pressing Enter (default choice)
	input := strings.NewReader("\n")
	spec, err := r.PromptQuality(input, profiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec != profiles[0].FormatSpec {
		t.Errorf("got spec %q; want default %q", spec, profiles[0].FormatSpec)
	}
}

func TestPromptQualityChoice(t *testing.T) {
	r := NewRenderer(ioDiscard{})
	profiles := config.QualityProfiles()

	// Simulating choosing '2'
	input := strings.NewReader("2\n")
	spec, err := r.PromptQuality(input, profiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec != profiles[1].FormatSpec {
		t.Errorf("got spec %q; want profile 2 %q", spec, profiles[1].FormatSpec)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
