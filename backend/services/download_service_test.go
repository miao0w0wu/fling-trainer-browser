package services

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadWritesFileIntoConfiguredRootAndReportsCompletion(t *testing.T) {
	root := t.TempDir()
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
	service.downloadRoot = func() string { return root }
	path, err := service.Download(server.URL+"/download", func(percent float64, _ string) {
		lastPercent = percent
	})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	expected := filepath.Join(root, "FLiNG_Trainers", "sample-trainer.zip")
	if path != expected {
		t.Fatalf("unexpected destination: got %s, want %s", path, expected)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Fatalf("downloaded content does not match source")
	}
	if lastPercent != 100 {
		t.Fatalf("expected final progress to be 100, got %v", lastPercent)
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Fatalf("temporary part file still exists")
	}
}

func TestDownloadAddsExecutableExtension(t *testing.T) {
	root := t.TempDir()
	payload := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 1024)...)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Disposition", `attachment; filename="fling-trainer"`)
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(payload)
	}))
	defer server.Close()

	service := NewDownloadService()
	service.downloadRoot = func() string { return root }
	path, err := service.Download(server.URL+"/downloads/12345", nil)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if filepath.Base(path) != "fling-trainer.exe" {
		t.Fatalf("expected executable extension, got: %s", filepath.Base(path))
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Fatalf("downloaded content does not match source")
	}
}

func TestDownloadReplacesHandlerExtension(t *testing.T) {
	root := t.TempDir()
	payload := append([]byte("MZ"), bytes.Repeat([]byte{0x00}, 512)...)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Disposition", `attachment; filename="download.php"`)
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(payload)
	}))
	defer server.Close()

	service := NewDownloadService()
	service.downloadRoot = func() string { return root }
	path, err := service.Download(server.URL+"/dl.php?id=1", nil)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if filepath.Base(path) != "download.exe" {
		t.Fatalf("expected handler extension to be replaced, got: %s", filepath.Base(path))
	}
}

func TestEnsureExtension(t *testing.T) {
	exe := append([]byte("MZ"), bytes.Repeat([]byte{0x00}, 64)...)
	zip := append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0x00}, 64)...)
	rar := append([]byte("Rar!\x1a\x07\x00"), bytes.Repeat([]byte{0x00}, 64)...)

	cases := []struct {
		name        string
		filename    string
		contentType string
		payload     []byte
		want        string
	}{
		{"appends exe when extension missing", "trainer", "", exe, "trainer.exe"},
		{"replaces web handler extension", "download.php", "application/octet-stream", exe, "download.exe"},
		{"keeps matching zip extension", "trainer.zip", "application/octet-stream", zip, "trainer.zip"},
		{"overrides zip name for exe payload", "trainer.zip", "application/octet-stream", exe, "trainer.exe"},
		{"uses content type when magic unknown", "trainer", "application/x-msdownload", []byte("plain data"), "trainer.exe"},
		{"uses content type for zip", "trainer", "application/zip", []byte("plain data"), "trainer.zip"},
		{"keeps real extension without magic", "notes.txt", "text/plain", []byte("hello"), "notes.txt"},
		{"keeps matching case-insensitively", "TRAINER.EXE", "", exe, "TRAINER.EXE"},
		{"detects rar magic", "package", "", rar, "package.rar"},
		{"keeps bare name when nothing detected", "mystery", "application/octet-stream", []byte("hello"), "mystery"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ensureExtension(testCase.filename, testCase.contentType, testCase.payload)
			if got != testCase.want {
				t.Fatalf("ensureExtension(%q, %q) = %q, want %q", testCase.filename, testCase.contentType, got, testCase.want)
			}
		})
	}
}
