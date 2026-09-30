package scraper

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"changeme/backend/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

var (
	gameVersionPattern = regexp.MustCompile(`(?i)game\s+version\s*[:：]\s*([^·|]+?)\s*(?:·|last\s+updated)`)
	lastUpdatedPattern = regexp.MustCompile(`(?i)last\s+updated\s*[:：]\s*([^·|]+?)(?:options|$)`)
)

// DetailParser retrieves and parses a trainer detail page.
type DetailParser struct {
	userAgent string
}

// NewDetailParser creates a detail parser using the same request policy as the
// search agent.
func NewDetailParser() *DetailParser {
	return &DetailParser{userAgent: randomUserAgent()}
}

// Parse fetches detailURL and returns the normalized trainer metadata.
func (p *DetailParser) Parse(detailURL string) (*models.TrainerDetail, error) {
	detailURL = strings.TrimSpace(detailURL)
	if detailURL == "" {
		return nil, errors.New("detail URL cannot be empty")
	}

	var detail *models.TrainerDetail
	var statusCode int
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		detail, statusCode, lastErr = p.parseOnce(detailURL)
		if lastErr == nil || !shouldRetry(statusCode, lastErr) {
			break
		}
	}
	if lastErr != nil {
		if statusCode > 0 {
			return nil, fmt.Errorf("fetch detail page (%d): %w", statusCode, lastErr)
		}
		return nil, fmt.Errorf("fetch detail page: %w", lastErr)
	}
	if detail == nil {
		return nil, errors.New("detail page did not contain an HTML document")
	}
	if detail.Title == "" {
		return nil, errors.New("detail page did not contain a trainer title")
	}
	return detail, nil
}

func (p *DetailParser) parseOnce(detailURL string) (*models.TrainerDetail, int, error) {
	var detail *models.TrainerDetail
	statusCode := 0
	userAgent := randomUserAgent()
	collector := colly.NewCollector(
		colly.UserAgent(userAgent),
		colly.ParseHTTPErrorResponse(),
	)
	collector.SetRequestTimeout(20 * time.Second)
	collector.Limit(&colly.LimitRule{
		DomainGlob: "*flingtrainer.com*",
		Delay:      500 * time.Millisecond,
	})
	collector.OnRequest(func(request *colly.Request) {
		request.Headers.Set("User-Agent", userAgent)
	})
	collector.OnResponse(func(response *colly.Response) {
		statusCode = response.StatusCode
	})
	collector.OnHTML("html", func(element *colly.HTMLElement) {
		detail = parseDetail(element.DOM, element.Request.AbsoluteURL(""))
	})

	err := collector.Visit(detailURL)
	return detail, statusCode, err
}

func parseDetail(document *goquery.Selection, sourceURL string) *models.TrainerDetail {
	content := document.Find(".entry-content, .entry").First()
	if content.Length() == 0 {
		content = document
	}

	detail := &models.TrainerDetail{
		Title:     cleanText(document.Find("h1.entry-title, h1.post-title").First().Text()),
		SourceURL: sourceURL,
	}
	detail.GameVersion = detailMatch(content.Text(), gameVersionPattern.String())
	detail.LastUpdated = detailMatch(content.Text(), lastUpdatedPattern.String())
	detail.Options = parseOptions(content)
	detail.Images = parseImages(content, sourceURL)
	detail.DownloadURL = parseDownloadURL(content, sourceURL)
	detail.Description = parseDescription(content)

	if detail.Title == "" {
		detail.Title = cleanText(document.Find("title").First().Text())
	}
	return detail
}

func parseOptions(content *goquery.Selection) []string {
	var options []string
	content.Find("ul li").Each(func(_ int, selection *goquery.Selection) {
		text := selection.Clone().Find("script, style").Remove().End().Text()
		text = cleanText(text)
		if text != "" && !containsAny(strings.ToLower(text), "download", "comment") {
			options = appendUniqueString(options, text)
		}
	})
	if len(options) > 0 {
		return options
	}

	content.Find("p").Each(func(_ int, selection *goquery.Selection) {
		text := selection.Clone().Find("script, style").Remove().End().Text()
		for _, line := range strings.Split(text, "\n") {
			line = cleanText(line)
			if strings.Contains(line, "–") || strings.Contains(line, "-") {
				if strings.HasPrefix(line, "Num ") || strings.HasPrefix(line, "Ctrl+") ||
					strings.HasPrefix(line, "PageUp") || strings.HasPrefix(line, "PageDown") {
					options = appendUniqueString(options, line)
				}
			}
		}
	})
	return options
}

func parseImages(content *goquery.Selection, sourceURL string) []string {
	var images []string
	content.Find("img").Each(func(_ int, selection *goquery.Selection) {
		src, exists := selection.Attr("src")
		if !exists || selection.HasClass("attachment-icon") || isExcludedImage(src) {
			return
		}
		images = appendUniqueString(images, absoluteURL(sourceURL, src))
	})
	return images
}

func parseDownloadURL(content *goquery.Selection, sourceURL string) string {
	var downloadURL string
	content.Find("a[href]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href, exists := selection.Attr("href")
		if !exists {
			return true
		}
		lowerHref := strings.ToLower(href)
		if strings.HasSuffix(strings.Split(lowerHref, "?")[0], ".zip") ||
			strings.Contains(lowerHref, "/uploads/") ||
			strings.Contains(lowerHref, "/downloads/") {
			downloadURL = absoluteURL(sourceURL, href)
			return false
		}
		return true
	})
	return downloadURL
}

func parseDescription(content *goquery.Selection) string {
	var paragraphs []string
	content.Find("p").Each(func(_ int, selection *goquery.Selection) {
		text := cleanText(selection.Text())
		if text == "" || strings.Contains(strings.ToLower(text), "game version:") ||
			strings.Contains(strings.ToLower(text), "last updated:") ||
			isOptionText(text) {
			return
		}
		paragraphs = append(paragraphs, text)
	})
	return truncateRunes(strings.Join(paragraphs, "\n\n"), 500)
}

func detailMatch(value, expression string) string {
	matches := regexp.MustCompile(expression).FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return cleanText(matches[1])
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func absoluteURL(sourceURL, href string) string {
	request, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil || request.URL == nil {
		return href
	}
	relative, err := request.URL.Parse(href)
	if err != nil {
		return href
	}
	return relative.String()
}

func isExcludedImage(src string) bool {
	lowerSrc := strings.ToLower(src)
	return containsAny(lowerSrc, "icon", "logo", "gravatar", "avatar", "tooltip")
}

func isOptionText(value string) bool {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lowerValue, "num ") ||
		strings.HasPrefix(lowerValue, "ctrl+") ||
		strings.HasPrefix(lowerValue, "pageup") ||
		strings.HasPrefix(lowerValue, "pagedown")
}

func containsAny(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func appendUniqueString(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
