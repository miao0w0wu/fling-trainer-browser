package services

import (
	"errors"
	"strings"

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
}

// NewTrainerService creates the application-facing trainer service.
func NewTrainerService() *TrainerService {
	return &TrainerService{
		searchAgent:     scraper.NewSearchAgent(),
		detailParser:    scraper.NewDetailParser(),
		downloadService: NewDownloadService(),
	}
}

// SearchTrainer searches FLiNG for trainers matching gameName.
func (s *TrainerService) SearchTrainer(gameName string) ([]models.SearchResult, error) {
	if strings.TrimSpace(gameName) == "" {
		return nil, errors.New("game name cannot be empty")
	}
	return s.searchAgent.Search(gameName)
}

// GetTrainerDetail parses a trainer detail page.
func (s *TrainerService) GetTrainerDetail(detailURL string) (*models.TrainerDetail, error) {
	if strings.TrimSpace(detailURL) == "" {
		return nil, errors.New("detail URL cannot be empty")
	}
	return s.detailParser.Parse(detailURL)
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
