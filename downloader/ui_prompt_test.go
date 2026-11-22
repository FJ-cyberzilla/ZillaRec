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

        runPromptQualityTest(t, r, profiles, "\n", profiles[0].FormatSpec)
}

func TestPromptQualityChoice(t *testing.T) {
        r := NewRenderer(ioDiscard{})
        profiles := config.QualityProfiles()

        runPromptQualityTest(t, r, profiles, profiles[1].ID+"\n", profiles[1].FormatSpec)
}

func runPromptQualityTest(
        t *testing.T,
        r *Renderer,
        profiles []config.QualityOption,
        rawInput string,
        wantSpec string,
) {
        t.Helper()

        spec, err := r.PromptQuality(strings.NewReader(rawInput), profiles)
        if err != nil {
                t.Fatalf("PromptQuality(%q) unexpected error: %v", rawInput, err)
        }
        if spec != wantSpec {
                t.Errorf("PromptQuality(%q) = %q; want %q", rawInput, spec, wantSpec)
        }
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
        return len(p), nil
}
