package scraper

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"changeme/backend/models"
	"github.com/gocolly/colly/v2"
)

const (
	targetBaseURL   = "https://flingtrainer.com"
	searchUserAgent = "FLiNG-Trainer-Browser/0.1 (+https://flingtrainer.com/)"
)

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// SearchAgent searches FLiNG using a direct trainer URL first and WordPress
// search as a fallback.
type SearchAgent struct {
	userAgent string
}

// NewSearchAgent creates a search agent with the required request policy.
func NewSearchAgent() *SearchAgent {
	return &SearchAgent{userAgent: searchUserAgent}
}

// Search searches the direct trainer route and falls back to WordPress search
// when the direct route is unavailable or produces no result.
func (a *SearchAgent) Search(gameName string) ([]models.SearchResult, error) {
	query := strings.TrimSpace(gameName)
	if query == "" {
		return nil, errors.New("game name cannot be empty")
	}

	directResults, directStatus, directErr := a.searchDirect(query)
	if directErr == nil && directStatus != http.StatusNotFound && len(directResults) > 0 {
		return directResults, nil
	}

	fallbackResults, fallbackErr := a.searchWordPress(query)
	if fallbackErr != nil {
		if directErr != nil {
			return nil, fmt.Errorf("direct search failed: %w; fallback search failed: %v", directErr, fallbackErr)
		}
		return nil, fallbackErr
	}
	return fallbackResults, nil
}

func (a *SearchAgent) searchDirect(query string) ([]models.SearchResult, int, error) {
	var results []models.SearchResult
	statusCode := http.StatusOK
	collector := a.newCollector()
	collector.OnResponse(func(response *colly.Response) {
		statusCode = response.StatusCode
	})
	collector.OnError(func(response *colly.Response, _ error) {
		if response != nil {
			statusCode = response.StatusCode
		}
	})
	collector.OnHTML("h1.entry-title, article", func(element *colly.HTMLElement) {
		result := resultFromElement(element)
		if result.URL == "" {
			result.URL = element.Request.URL.String()
		}
		if result.Title != "" {
			results = appendUniqueResult(results, result)
		}
	})

	err := collector.Visit(directURL(query))
	if err != nil {
		return nil, statusCode, err
	}
	return results, statusCode, nil
}

func (a *SearchAgent) searchWordPress(query string) ([]models.SearchResult, error) {
	var results []models.SearchResult
	collector := a.newCollector()
	collector.OnHTML("article, .search-result, .post", func(element *colly.HTMLElement) {
		result := resultFromElement(element)
		if result.Title != "" && result.URL != "" {
			results = appendUniqueResult(results, result)
		}
	})
	collector.OnHTML("a[href]", func(element *colly.HTMLElement) {
		href := element.Request.AbsoluteURL(element.Attr("href"))
		title := strings.TrimSpace(element.Text)
		if title == "" || !strings.Contains(strings.ToLower(href), "flingtrainer.com") {
			return
		}
		if strings.Contains(strings.ToLower(href), "/trainer/") || strings.Contains(strings.ToLower(title), "trainer") {
			results = appendUniqueResult(results, models.SearchResult{
				Title: title,
				URL:   href,
			})
		}
	})

	if err := collector.Visit(wordPressSearchURL(query)); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *SearchAgent) newCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent(a.userAgent),
		colly.MaxDepth(1),
		colly.ParseHTTPErrorResponse(),
	)
	collector.SetRequestTimeout(20 * time.Second)
	collector.SetRedirectHandler(func(request *http.Request, _ []*http.Request) error {
		request.Header.Set("User-Agent", a.userAgent)
		return nil
	})
	if err := collector.Limit(&colly.LimitRule{
		DomainGlob: "*flingtrainer.com*",
		Delay:      500 * time.Millisecond,
	}); err != nil {
		panic(fmt.Sprintf("configure scraper rate limit: %v", err))
	}
	collector.OnRequest(func(request *colly.Request) {
		request.Headers.Set("User-Agent", a.userAgent)
	})
	return collector
}

func directURL(query string) string {
	return targetBaseURL + "/trainer/" + slugify(query) + "-trainer/"
}

func wordPressSearchURL(query string) string {
	return targetBaseURL + "/?s=" + url.QueryEscape(query)
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlugCharacters.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func resultFromElement(element *colly.HTMLElement) models.SearchResult {
	title := strings.TrimSpace(element.ChildText("h1, h2, h3, h4, .entry-title, .post-title, a"))
	href := strings.TrimSpace(element.ChildAttr("h1 a, h2 a, h3 a, h4 a, .entry-title a, .post-title a, a", "href"))
	if href != "" {
		href = element.Request.AbsoluteURL(href)
	}
	text := strings.Join(strings.Fields(element.Text), " ")
	return models.SearchResult{
		Title:   title,
		URL:     href,
		Version: firstMatch(text, `(?i)(?:version|ver(?:sion)?)\s*[:：]?\s*([0-9][\w.-]*)`),
		Updated: firstMatch(text, `(?i)(?:updated|update|last updated)\s*[:：]?\s*([A-Za-z0-9,./ -]+)`),
		Options: firstMatch(text, `(?i)(?:options?|features?)\s*[:：]?\s*(.+)`),
	}
}

func firstMatch(value, expression string) string {
	matches := regexp.MustCompile(expression).FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func appendUniqueResult(results []models.SearchResult, candidate models.SearchResult) []models.SearchResult {
	for _, result := range results {
		if result.URL == candidate.URL && candidate.URL != "" {
			return results
		}
	}
	return append(results, candidate)
}
