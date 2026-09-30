package models

// SearchResult describes one trainer returned by the search agent.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Version string `json:"version"`
	Updated string `json:"updated"`
	Options string `json:"options"`
}

// TrainerDetail contains the metadata needed to render a trainer detail page.
type TrainerDetail struct {
	Title       string   `json:"title"`
	GameVersion string   `json:"gameVersion"`
	LastUpdated string   `json:"lastUpdated"`
	Options     []string `json:"options"`
	Images      []string `json:"images"`
	DownloadURL string   `json:"downloadUrl"`
	Description string   `json:"description"`
	SourceURL   string   `json:"sourceUrl"`
}
