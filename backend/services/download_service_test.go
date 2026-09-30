package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadWritesFileAndReportsCompletion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	payload := []byte(strings.Repeat("trainer-data-", 4096))
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("User-Agent") != downloadUserAgent {
			t.Errorf("unexpected user agent: %q", request.Header.Get("User-Agent"))
		}
		response.Header().Set("Content-Disposition", `attachment; filename="sample-trainer.zip"`)
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(payload)
	}))
	defer server.Close()

	var lastPercent float64
	service := NewDownloadService()
	path, err := service.Download(server.URL+"/download", func(percent float64, _ string) {
		lastPercent = percent
	})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(path))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != string(payload) {
		t.Fatalf("downloaded content does not match source")
	}
	if filepath.Base(path) != "sample-trainer.zip" {
		t.Fatalf("unexpected filename: %s", filepath.Base(path))
	}
	if lastPercent != 100 {
		t.Fatalf("expected final progress to be 100, got %v", lastPercent)
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Fatalf("temporary part file still exists")
	}
}
