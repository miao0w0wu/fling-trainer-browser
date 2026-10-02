package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	downloadUserAgent  = "FLiNG-Trainer-Browser/0.1 (+https://flingtrainer.com/)"
	downloadBufferSize = 32 * 1024
	progressInterval   = 200 * time.Millisecond
	sniffLength        = 512
)

// DownloadProgressCallback receives the current percentage and transfer speed.
type DownloadProgressCallback func(percent float64, speed string)

// DownloadService downloads trainer archives into the user's Downloads folder.
type DownloadService struct {
	client    *http.Client
	userAgent string
	// downloadRoot returns the user's configured downloads location; a field
	// so tests can point it at a temporary directory.
	downloadRoot func() string
}

// NewDownloadService creates a download service with a bounded HTTP client.
func NewDownloadService() *DownloadService {
	return &DownloadService{
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
		userAgent:    downloadUserAgent,
		downloadRoot: userDownloadsRoot,
	}
}

// Download saves the resource at downloadURL under the user's configured
// Downloads folder (the system setting on Windows) in FLiNG_Trainers.
// Progress callbacks are throttled to at most one callback every 200ms, with a
// final 100% callback always emitted after the file has been committed.
func (s *DownloadService) Download(downloadURL string, onProgress DownloadProgressCallback) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(downloadURL))
	if err != nil || parsedURL.Scheme != "http" && parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return "", errors.New("download URL must be an absolute HTTP or HTTPS URL")
	}

	downloadDir := filepath.Join(s.downloadRoot(), "FLiNG_Trainers")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}

	request, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	request.Header.Set("User-Agent", s.userAgent)

	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download trainer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download trainer: unexpected HTTP status %s", response.Status)
	}

	sniffed, err := readPayloadPrefix(response.Body)
	if err != nil {
		return "", fmt.Errorf("read downloaded trainer: %w", err)
	}

	filename := ensureExtension(filenameFromResponse(response, parsedURL), response.Header.Get("Content-Type"), sniffed)
	destination := filepath.Join(downloadDir, filename)
	temporaryPath := destination + ".part"
	file, err := os.Create(temporaryPath)
	if err != nil {
		return "", fmt.Errorf("create temporary download file: %w", err)
	}

	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(temporaryPath)
		}
	}()

	if _, err := file.Write(sniffed); err != nil {
		return "", fmt.Errorf("write downloaded trainer: %w", err)
	}

	totalBytes := response.ContentLength
	downloadedBytes := int64(len(sniffed))
	buffer := make([]byte, downloadBufferSize)
	lastCallback := time.Time{}
	startedAt := time.Now()

	for {
		bytesRead, readErr := response.Body.Read(buffer)
		if bytesRead > 0 {
			if _, err := file.Write(buffer[:bytesRead]); err != nil {
				return "", fmt.Errorf("write downloaded trainer: %w", err)
			}
			downloadedBytes += int64(bytesRead)
			now := time.Now()
			if onProgress != nil && (lastCallback.IsZero() || now.Sub(lastCallback) >= progressInterval) {
				onProgress(downloadPercent(downloadedBytes, totalBytes), transferSpeed(downloadedBytes, startedAt))
				lastCallback = now
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return "", fmt.Errorf("read downloaded trainer: %w", readErr)
			}
			break
		}
	}

	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("flush downloaded trainer: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close downloaded trainer: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return "", fmt.Errorf("commit downloaded trainer: %w", err)
	}
	success = true

	if onProgress != nil {
		onProgress(100, transferSpeed(downloadedBytes, startedAt))
	}
	return destination, nil
}

// OpenFolder opens the trainer download location in the OS file manager. When
// filePath names an existing file, the manager opens with that file selected;
// otherwise the FLiNG_Trainers folder is opened, created if missing.
func (s *DownloadService) OpenFolder(filePath string) error {
	if candidate := strings.TrimSpace(filePath); candidate != "" {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return openInFileManager(candidate)
		}
	}
	downloadDir := filepath.Join(s.downloadRoot(), "FLiNG_Trainers")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	return openInFileManager(downloadDir)
}

func filenameFromResponse(response *http.Response, parsedURL *url.URL) string {
	if disposition := response.Header.Get("Content-Disposition"); disposition != "" {
		_, parameters, err := mime.ParseMediaType(disposition)
		if err == nil {
			if filename := safeFilename(parameters["filename"]); filename != "" {
				return filename
			}
		}
	}

	if filename := safeFilename(filepath.Base(parsedURL.Path)); filename != "" && filename != "." {
		return filename
	}
	return fmt.Sprintf("trainer-%s.zip", time.Now().Format("20060102-150405"))
}

// readPayloadPrefix reads up to sniffLength leading bytes from the body so the
// payload type can be detected without consuming the rest of the stream.
func readPayloadPrefix(body io.Reader) ([]byte, error) {
	prefix := make([]byte, sniffLength)
	read, err := io.ReadFull(body, prefix)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return prefix[:read], nil
}

// ensureExtension makes sure the saved filename carries an extension matching
// the actual payload: download handlers often serve names like "download.php"
// or bare names without any extension, while the bytes are an executable or
// archive. Magic bytes win over the Content-Type header, which wins over the
// handler's name.
func ensureExtension(filename, contentType string, payload []byte) string {
	currentExtension := strings.ToLower(filepath.Ext(filename))
	if detected := extensionFromPayload(payload); detected != "" {
		if currentExtension == detected {
			return filename
		}
		return strings.TrimSuffix(filename, filepath.Ext(filename)) + detected
	}

	if currentExtension != "" && !webHandlerExtensions[currentExtension] {
		return filename
	}
	if detected := extensionFromContentType(contentType); detected != "" {
		return strings.TrimSuffix(filename, filepath.Ext(filename)) + detected
	}
	return filename
}

func extensionFromPayload(payload []byte) string {
	switch {
	case bytes.HasPrefix(payload, []byte("MZ")):
		return ".exe"
	case bytes.HasPrefix(payload, []byte("PK\x03\x04")):
		return ".zip"
	case bytes.HasPrefix(payload, []byte("Rar!\x1a\x07")):
		return ".rar"
	case bytes.HasPrefix(payload, []byte("7z\xbc\xaf\x27\x1c")):
		return ".7z"
	default:
		return ""
	}
}

var extensionsByContentType = map[string]string{
	"application/x-msdownload":                      ".exe",
	"application/x-msdos-program":                   ".exe",
	"application/vnd.microsoft.portable-executable": ".exe",
	"application/zip":                               ".zip",
	"application/vnd.rar":                           ".rar",
	"application/x-rar-compressed":                  ".rar",
	"application/x-7z-compressed":                   ".7z",
}

func extensionFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return extensionsByContentType[strings.ToLower(mediaType)]
}

// webHandlerExtensions are server-side script names a download link may end
// with; they never describe the payload and are replaced when a real type is
// detected.
var webHandlerExtensions = map[string]bool{
	".php":  true,
	".asp":  true,
	".aspx": true,
	".ashx": true,
	".asmx": true,
	".jsp":  true,
	".cgi":  true,
	".do":   true,
	".htm":  true,
	".html": true,
}

var invalidFilenameCharacters = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

func safeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = invalidFilenameCharacters.ReplaceAllString(filename, "_")
	filename = strings.Trim(filename, ". ")
	if filename == "" || filename == "." || filename == ".." {
		return ""
	}
	return filename
}

func downloadPercent(downloaded, total int64) float64 {
	if total <= 0 {
		return 0
	}
	percent := float64(downloaded) / float64(total) * 100
	if percent > 100 {
		return 100
	}
	return percent
}

func transferSpeed(downloaded int64, startedAt time.Time) string {
	elapsed := time.Since(startedAt).Seconds()
	if elapsed <= 0 {
		return "0.00 MB/s"
	}
	return fmt.Sprintf("%.2f MB/s", float64(downloaded)/elapsed/1024/1024)
}
