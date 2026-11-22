package omnicharge

import (
	"encoding/json"
	"os"
	"time"
)

// TagHandler manages metadata tagging for downloaded media.
type TagHandler struct {
	AppName string
	Brand   string
}

// MetadataSidecar represents the JSON structure of the metadata sidecar file.
type MetadataSidecar struct {
	AppName string    `json:"app_name"`
	Brand   string    `json:"brand"`
	Date    time.Time `json:"date"`
	Format  string    `json:"format"`
}

// NewTagHandler creates a new TagHandler instance.
func NewTagHandler(appName, brand string) *TagHandler {
	return &TagHandler{
		AppName: appName,
		Brand:   brand,
	}
}

// WriteSidecarMetadata writes a JSON sidecar file containing metadata for the given file path.
func (th *TagHandler) WriteSidecarMetadata(filePath string, format string) error {
	meta := MetadataSidecar{
		AppName: th.AppName,
		Brand:   th.Brand,
		Date:    time.Now().UTC(),
		Format:  format,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	sidecarPath := filePath + ".meta.json"
	return os.WriteFile(sidecarPath, data, 0644)
}
