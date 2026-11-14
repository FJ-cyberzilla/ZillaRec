package downloader

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseProgressTemplate(t *testing.T) {
	tests := []struct {
		input            string
		expectedPercent  float64
		expectedSpeed    string
		expectedETA      string
		expectedTitle    string
		expectedDuration string
	}{
		{
			input:            "45.5%\t1.2MiB/s\t02:15\tTest Video Title\t03:45",
			expectedPercent:  45.5,
			expectedSpeed:    "1.2MiB/s",
			expectedETA:      "02:15",
			expectedTitle:    "Test Video Title",
			expectedDuration: "03:45",
		},
		{
			input:            "N/A\tN/A\tN/A\tN/A\tN/A",
			expectedPercent:  0,
			expectedSpeed:    "",
			expectedETA:      "",
			expectedTitle:    "",
			expectedDuration: "",
		},
		{
			input:            "100%\t500KiB/s\t00:00\t",
			expectedPercent:  100.0,
			expectedSpeed:    "500KiB/s",
			expectedETA:      "00:00",
			expectedTitle:    "",
			expectedDuration: "",
		},
	}

	for _, tc := range tests {
		p := parseProgressTemplate(tc.input)
		if p.Percent != tc.expectedPercent {
			t.Errorf("parseProgressTemplate(%q) Percent = %f, expected %f", tc.input, p.Percent, tc.expectedPercent)
		}
		if p.Speed != tc.expectedSpeed {
			t.Errorf("parseProgressTemplate(%q) Speed = %q, expected %q", tc.input, p.Speed, tc.expectedSpeed)
		}
		if p.ETA != tc.expectedETA {
			t.Errorf("parseProgressTemplate(%q) ETA = %q, expected %q", tc.input, p.ETA, tc.expectedETA)
		}
		if p.Title != tc.expectedTitle {
			t.Errorf("parseProgressTemplate(%q) Title = %q, expected %q", tc.input, p.Title, tc.expectedTitle)
		}
		if p.Duration != tc.expectedDuration {
			t.Errorf("parseProgressTemplate(%q) Duration = %q, expected %q", tc.input, p.Duration, tc.expectedDuration)
		}
	}
}

func TestDedupeAbs(t *testing.T) {
	paths := []string{"/abs/path/video1.mp4", "/abs/path/video1.mp4", "video2.mp4"}
	deduped := dedupeAbs(paths)
	if len(deduped) != 2 {
		t.Errorf("dedupeAbs expected 2 unique paths, got %d (%v)", len(deduped), deduped)
	}
}

func TestHandleLine(t *testing.T) {
	var progressCalled, destCalled, mergingCalled, logCalled bool
	cb := Callbacks{
		OnProgress: func(p Progress) {
			progressCalled = true
		},
		OnDestination: func(dest string) {
			destCalled = true
		},
		OnMerging: func() {
			mergingCalled = true
		},
		OnLog: func(line string) {
			logCalled = true
		},
	}

	var dests []string

	handleLine("download:50.0%\t1MiB/s\t01:00\tTitle\t02:00", cb, &dests)
	if !progressCalled {
		t.Errorf("expected OnProgress to be called")
	}

	handleLine("[download] Destination: /path/to/video.mp4", cb, &dests)
	if !destCalled || len(dests) != 1 || dests[0] != "/path/to/video.mp4" {
		t.Errorf("expected destination parsed correctly, got %v", dests)
	}

	handleLine(`[Merger] Merging formats into "/path/to/video.mp4"`, cb, &dests)
	if !mergingCalled {
		t.Errorf("expected OnMerging to be called")
	}

	handleLine("some log message", cb, &dests)
	if !logCalled {
		t.Errorf("expected OnLog to be called")
	}
}

func TestProgressBarRendering(t *testing.T) {
	var buf bytes.Buffer
	r := &Renderer{out: &buf, color: colorizer{enabled: false}}
	bar := NewProgressBar(r)

	bar.SetStatus("Initializing download...")
	if !strings.Contains(buf.String(), "status: Initializing download...") {
		t.Errorf("expected uncolored status line, got %q", buf.String())
	}

	buf.Reset()
	bar.lastDraw = time.Time{}
	bar.Update(Progress{Percent: 50.0, Speed: "2MiB/s", ETA: "00:10"})
	if !strings.Contains(buf.String(), "progress: 50.0%") {
		t.Errorf("expected uncolored progress line, got %q", buf.String())
	}

	bar.EndProgress()
}
