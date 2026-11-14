package downloader

import (
	"fmt"
	"strings"
)

// AnalyzeError inspects raw errors and yt-dlp stderr output to produce
// clear, actionable system alerts for network, DNS, VPN/proxy, and YouTube blocks.
func AnalyzeError(err error, stderr string) error {
	if err == nil {
		return nil
	}

	combined := err.Error() + "\n" + stderr
	lower := strings.ToLower(combined)

	// 1. YouTube IP Ban / Rate Limit / Bot Detection / Geo-Restriction
	if strings.Contains(lower, "429") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "sign in to confirm") ||
		strings.Contains(lower, "bot detection") ||
		strings.Contains(lower, "unavailable") ||
		strings.Contains(lower, "private") ||
		strings.Contains(lower, "geo-restricted") ||
		strings.Contains(lower, "blocked") ||
		strings.Contains(lower, "copyright") {
		return fmt.Errorf("[YOUTUBE RESTRICTION / IP BAN ALERT]: YouTube has blocked or rate-limited your IP (HTTP 429 / Bot Detection), or the video is geo-restricted/unavailable.\n\nTroubleshooting:\n• Enable or switch your VPN region/server.\n• Add valid browser cookies to cookies.txt (--cookies).\n• Ensure yt-dlp is up to date.\n\nRaw details: %w", err)
	}

	// 2. DNS Problems
	if strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "servfail") ||
		strings.Contains(lower, "nxdomain") ||
		strings.Contains(lower, "temporary failure in name resolution") {
		return fmt.Errorf("[DNS ERROR ALERT]: DNS resolution failed. Cannot resolve hostname.\n\nTroubleshooting:\n• Check your internet connection or Wi-Fi.\n• Check if your VPN DNS settings are misconfigured.\n• Try switching DNS provider (e.g. 1.1.1.1 or 8.8.8.8).\n\nRaw details: %w", err)
	}

	// 3. No Internet / Network Unreachable
	if strings.Contains(lower, "network is unreachable") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "network down") {
		return fmt.Errorf("[NETWORK ERROR ALERT]: No internet connection or network unreachable.\n\nTroubleshooting:\n• Turn on Wi-Fi or mobile data.\n• Verify your device has active internet access.\n\nRaw details: %w", err)
	}

	// 4. VPN / Proxy Issues
	if strings.Contains(lower, "proxy") ||
		strings.Contains(lower, "socks") ||
		strings.Contains(lower, "tunnel connection failed") ||
		strings.Contains(lower, "handshake failure") ||
		strings.Contains(lower, "connection reset by peer") {
		return fmt.Errorf("[VPN / PROXY ERROR ALERT]: VPN connection failure or proxy issue detected.\n\nTroubleshooting:\n• Check if your VPN is connected or if the proxy server is online.\n• Restart your VPN application or switch servers.\n\nRaw details: %w", err)
	}

	// 5. Default Fallback Error with context
	if stderr != "" {
		return fmt.Errorf("[DOWNLOAD ERROR]: %w\n\nDiagnostic Output:\n%s", err, strings.TrimSpace(stderr))
	}

	return err
}
