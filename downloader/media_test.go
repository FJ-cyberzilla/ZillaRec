package downloader

import (
	"testing"
)

func TestRegisterWithMediaStoreEmpty(t *testing.T) {
	// Test that registering empty paths does not panic or error
	RegisterWithMediaStore(nil)
	RegisterWithMediaStore([]string{})
}
