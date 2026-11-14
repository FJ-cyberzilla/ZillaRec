package config

import "testing"

func TestQualityProfiles(t *testing.T) {
	profiles := QualityProfiles()
	if len(profiles) == 0 {
		t.Fatalf("expected quality profiles, got 0")
	}
	for _, p := range profiles {
		if p.ID == "" || p.Label == "" || p.FormatSpec == "" {
			t.Errorf("invalid profile: %+v", p)
		}
	}
}
