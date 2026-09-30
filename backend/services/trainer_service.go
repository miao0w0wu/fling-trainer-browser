package services

import (
	"errors"
	"strings"
	"sync"
	"time"

	"changeme/backend/models"
	"changeme/backend/scraper"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const downloadProgressEvent = "download:progress"

// DownloadProgress is emitted while a trainer archive is being downloaded.
type DownloadProgress struct {
	Percent float64 `json:"percent"`
	Speed   string  `json:"speed"`
}

// TrainerService exposes trainer search, detail parsing, and downloading to
// the Wails frontend.
type TrainerService struct {
	searchAgent     *scraper.SearchAgent
	detailParser    *scraper.DetailParser
	downloadService *DownloadService
	cacheMu         sync.RWMutex
	searchCache     map[string]searchCacheEntry
	detailCache     map[string]detailCacheEntry
}

type searchCacheEntry struct {
	results   []models.SearchResult
	expiresAt time.Time
}

type detailCacheEntry struct {
	detail    *models.TrainerDetail
	expiresAt time.Time
}

// NewTrainerService creates the application-facing trainer service.
func NewTrainerService() *TrainerService {
	return &TrainerService{
		searchAgent:     scraper.NewSearchAgent(),
		detailParser:    scraper.NewDetailParser(),
		downloadService: NewDownloadService(),
		searchCache:     make(map[string]searchCacheEntry),
		detailCache:     make(map[string]detailCacheEntry),
	}
}

// SearchTrainer searches FLiNG for trainers matching gameName.
func (s *TrainerService) SearchTrainer(gameName string) ([]models.SearchResult, error) {
	query := strings.TrimSpace(gameName)
	if query == "" {
		return nil, errors.New("game name cannot be empty")
	}
	s.cacheMu.RLock()
	cached, ok := s.searchCache[strings.ToLower(query)]
	s.cacheMu.RUnlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.results, nil
	}

	results, err := s.searchAgent.Search(query)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	s.searchCache[strings.ToLower(query)] = searchCacheEntry{
		results: results, expiresAt: time.Now().Add(10 * time.Minute),
	}
	s.cacheMu.Unlock()
	return results, nil
}

// GetTrainerDetail parses a trainer detail page.
func (s *TrainerService) GetTrainerDetail(detailURL string) (*models.TrainerDetail, error) {
	sourceURL := strings.TrimSpace(detailURL)
	if sourceURL == "" {
		return nil, errors.New("detail URL cannot be empty")
	}
	s.cacheMu.RLock()
	cached, ok := s.detailCache[sourceURL]
	s.cacheMu.RUnlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.detail, nil
	}

	detail, err := s.detailParser.Parse(sourceURL)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	s.detailCache[sourceURL] = detailCacheEntry{
		detail: detail, expiresAt: time.Now().Add(30 * time.Minute),
	}
	s.cacheMu.Unlock()
	return detail, nil
}

// DownloadTrainer downloads a trainer archive and emits download progress.
func (s *TrainerService) DownloadTrainer(downloadURL string) (string, error) {
	if strings.TrimSpace(downloadURL) == "" {
		return "", errors.New("download URL cannot be empty")
	}
	return s.downloadService.Download(downloadURL, func(percent float64, speed string) {
		application.Get().Event.Emit(downloadProgressEvent, DownloadProgress{
			Percent: percent,
			Speed:   speed,
		})
	})
}
