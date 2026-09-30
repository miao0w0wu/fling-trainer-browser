package services

import (
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
)

// DownloadProgressCallback receives the current percentage and transfer speed.
type DownloadProgressCallback func(percent float64, speed string)

// DownloadService downloads trainer archives into the user's Downloads folder.
type DownloadService struct {
	client    *http.Client
	userAgent string
}

// NewDownloadService creates a download service with a bounded HTTP client.
func NewDownloadService() *DownloadService {
	return &DownloadService{
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
		userAgent: downloadUserAgent,
	}
}

// Download saves the resource at downloadURL under ~/Downloads/FLiNG_Trainers.
// Progress callbacks are throttled to at most one callback every 200ms, with a
// final 100% callback always emitted after the file has been committed.
func (s *DownloadService) Download(downloadURL string, onProgress DownloadProgressCallback) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(downloadURL))
	if err != nil || parsedURL.Scheme != "http" && parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return "", errors.New("download URL must be an absolute HTTP or HTTPS URL")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	downloadDir := filepath.Join(homeDir, "Downloads", "FLiNG_Trainers")
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

	filename := filenameFromResponse(response, parsedURL)
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

	totalBytes := response.ContentLength
	var downloadedBytes int64
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
