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
			ID:         "best",
			Label:      "Best Available",
			FormatSpec: "bestvideo+bestaudio/best",
		},
		{
			ID:         "1080p",
			Label:      "1080p Full HD",
			FormatSpec: "bestvideo[height<=1080]+bestaudio/bestvideo[width<=1080]+bestaudio/best[height<=1080]/best",
		},
		{
			ID:         "720p",
			Label:      "720p HD",
			FormatSpec: "bestvideo[height<=720]+bestaudio/best[height<=720]/best",
		},
		{
			ID:         "480p",
			Label:      "480p SD",
			FormatSpec: "bestvideo[height<=480]+bestaudio/best[height<=480]/best",
		},
	}
}
