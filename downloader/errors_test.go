package downloader

import (
	"errors"
	"strings"
	"testing"
)

func TestAnalyzeError_Network(t *testing.T) {
	err := errors.New("dial tcp: lookup youtube.com: no such host")
	analyzed := AnalyzeError(err, "")
	if !strings.Contains(analyzed.Error(), "[DNS ERROR ALERT]") {
		t.Errorf("expected DNS error alert, got: %v", analyzed)
	}

	err2 := errors.New("dial tcp 192.168.1.1:443: connect: network is unreachable")
	analyzed2 := AnalyzeError(err2, "")
	if !strings.Contains(analyzed2.Error(), "[NETWORK ERROR ALERT]") {
		t.Errorf("expected network error alert, got: %v", analyzed2)
	}
}

func TestAnalyzeError_VPNProxy(t *testing.T) {
	err := errors.New("SOCKS5 proxy handshake failed: connection reset by peer")
	analyzed := AnalyzeError(err, "proxy connection timeout")
	if !strings.Contains(analyzed.Error(), "[VPN / PROXY ERROR ALERT]") {
		t.Errorf("expected VPN/proxy error alert, got: %v", analyzed)
	}
}

func TestAnalyzeError_YouTubeRestriction(t *testing.T) {
	err := errors.New("ERROR: [youtube] 12345: HTTP Error 429: Too Many Requests")
	analyzed := AnalyzeError(err, "Sign in to confirm you’re not a bot")
	if !strings.Contains(analyzed.Error(), "[YOUTUBE RESTRICTION / IP BAN ALERT]") {
		t.Errorf("expected YouTube restriction alert, got: %v", analyzed)
	}
}
