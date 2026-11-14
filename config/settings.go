package config

type Config struct {
	CookieFile string
	OutputDir  string
}

type QualityOption struct {
	ID         string
	Label      string
	FormatSpec string
}

func LoadDefaultConfig() Config {
	return Config{
		CookieFile: "cookies.txt",
		OutputDir:  "",
	}
}

func QualityProfiles() []QualityOption {
	return []QualityOption{
		{
			ID:         "1",
			Label:      "720p (Fast & low bandwidth)",
			FormatSpec: "bestvideo[height<=720]+bestaudio/best[height<=720]/best",
		},
		{
			ID:         "2",
			Label:      "1080p HD (landscape & shorts)",
			FormatSpec: "bestvideo[height<=1080]+bestaudio/bestvideo[width<=1080]+bestaudio/best",
		},
		{
			ID:         "3",
			Label:      "4K Ultra HD",
			FormatSpec: "bestvideo[height<=2160]+bestaudio/bestvideo[width<=2160]+bestaudio/best",
		},
	}
}
