package config

import "fmt"

// Config holds global runtime parameters for the downloader
type Config struct {
	PrimaryDNS    string
	SecondaryDNS  string
	CookieFile    string
	DefaultFormat string
}

// QualityOption defines predefined target video download profiles
type QualityOption struct {
	ID         string
	Label      string
	FormatSpec string
}

// LoadDefaultConfig instantiates default settings optimized for Termux execution
func LoadDefaultConfig() *Config {
	return &Config{
		PrimaryDNS:    "9.9.9.9:53",
		SecondaryDNS:  "149.112.112.112:53",
		CookieFile:    "cookies.txt",
		DefaultFormat: "bestvideo[height<=720]+bestaudio/best[height<=720]/best",
	}
}

// GetQualityProfiles returns flexible format selectors supporting landscape & vertical Shorts
func GetQualityProfiles() []QualityOption {
	return []QualityOption{
		{
			ID:         "1",
			Label:      "720p (Fast & Low Bandwidth)",
			FormatSpec: "bestvideo[height<=720]+bestaudio/best[height<=720]/best",
		},
		{
			ID:         "2",
			Label:      "1080p HD (Supports Landscape & Shorts)",
			FormatSpec: "bestvideo[height<=1080]+bestaudio/bestvideo[width<=1080]+bestaudio/best",
		},
		{
			ID:         "3",
			Label:      "4K Ultra HD",
			FormatSpec: "bestvideo[height<=2160]+bestaudio/bestvideo[width<=2160]+bestaudio/best",
		},
	}
}

// PrintConfig displays current environment settings for diagnostics
func (c *Config) PrintConfig() {
	fmt.Printf("[Config] DNS: %s | Cookie Path: %s\n", c.PrimaryDNS, c.CookieFile)
}
