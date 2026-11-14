package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"zillarec/omnicharge"
	"zillarec/storage"
)

func TestHandleStatus(t *testing.T) {
	// Initialize masterCtrl for testing status
	outDir, err := storage.OutputDir()
	if err != nil {
		t.Fatalf("failed to get output dir: %v", err)
	}
	masterCtrl, err = omnicharge.NewMasterController(outDir, 3)
	if err != nil {
		t.Fatalf("failed to init master controller: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handleStatus(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var snapshot omnicharge.ControllerSnapshot
	if err := json.NewDecoder(res.Body).Decode(&snapshot); err != nil {
		t.Fatalf("failed to decode status snapshot: %v", err)
	}

	if snapshot.State != omnicharge.StateIdle {
		t.Errorf("expected state 'IDLE', got '%s'", snapshot.State)
	}
}

func TestHandleExitMethod(t *testing.T) {
	// GET should be MethodNotAllowed
	req := httptest.NewRequest(http.MethodGet, "/api/exit", nil)
	rec := httptest.NewRecorder()
	handleExit(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET on /api/exit, got %d", rec.Result().StatusCode)
	}
}

func TestHandleDownloadInvalidPayload(t *testing.T) {
	// Invalid JSON payload should return 400 Bad Request
	reqBody := []byte(`{"invalid_json":`)
	req := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handleDownload(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid download request, got %d", rec.Result().StatusCode)
	}

	// Empty URL should return 400 Bad Request
	emptyReqBody, _ := json.Marshal(DownloadRequest{URL: ""})
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewReader(emptyReqBody))
	recEmpty := httptest.NewRecorder()

	handleDownload(recEmpty, reqEmpty)

	if recEmpty.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty URL download request, got %d", recEmpty.Result().StatusCode)
	}
}

func TestWebFilesEmbedded(t *testing.T) {
	// Verify that index.html is correctly embedded and accessible via FS
	subFS, err := subFSForTest()
	if err != nil {
		t.Fatalf("failed to get subFS: %v", err)
	}

	f, err := subFS.Open("index.html")
	if err != nil {
		t.Fatalf("failed to open embedded index.html: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to stat embedded index.html: %v", err)
	}

	if stat.Size() == 0 {
		t.Errorf("embedded index.html is empty")
	}
}

func subFSForTest() (http.FileSystem, error) {
	sub, err := http.FS(webFiles), error(nil)
	return sub, err
}
