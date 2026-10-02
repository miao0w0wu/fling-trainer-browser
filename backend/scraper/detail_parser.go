package scraper

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
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
	detail.DownloadOptions = parseDownloadOptions(content, sourceURL)
	for _, option := range detail.DownloadOptions {
		if option.URL != "" {
			detail.DownloadURL = option.URL
			break
		}
	}
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

// parseDownloadOptions collects every file listed in the page's Download
// section. On flingtrainer.com this is an attachments table where each row is
// one file and rows without links are group headings such as
// "Auto-Updating Version:" or "Standalone Versions:".
func parseDownloadOptions(content *goquery.Selection, sourceURL string) []models.DownloadOption {
	group := ""
	var options []models.DownloadOption
	content.Find("tr").Each(func(_ int, row *goquery.Selection) {
		link := downloadLinkFromRow(row)
		if link.Length() == 0 {
			if heading := groupHeading(row); heading != "" {
				group = heading
			}
			return
		}
		href, exists := link.Attr("href")
		if !exists || strings.TrimSpace(href) == "" {
			return
		}
		cells := row.Find("td")
		options = appendUniqueOption(options, models.DownloadOption{
			Group:     group,
			Name:      downloadName(link, href),
			URL:       absoluteURL(sourceURL, href),
			DateAdded: cellText(cells, "attachment-date", 1),
			FileSize:  cellText(cells, "attachment-size", 2),
			Downloads: cellText(cells, "attachment-downloads", 3),
		})
	})
	if len(options) == 0 {
		return scanDownloadLinks(content, sourceURL)
	}
	return options
}

func downloadLinkFromRow(row *goquery.Selection) *goquery.Selection {
	if link := row.Find("a.attachment-link").First(); link.Length() > 0 {
		return link
	}
	return row.Find("a[href]").FilterFunction(func(_ int, link *goquery.Selection) bool {
		href, exists := link.Attr("href")
		return exists && isDownloadHref(href)
	}).First()
}

func isDownloadHref(href string) bool {
	lowerHref := strings.ToLower(href)
	return strings.HasSuffix(strings.Split(lowerHref, "?")[0], ".zip") ||
		strings.Contains(lowerHref, "/uploads/") ||
		strings.Contains(lowerHref, "/downloads/")
}

// groupHeading returns the section title of a heading row such as
// "Auto-Updating Version:" and "" for anything that is not one.
func groupHeading(row *goquery.Selection) string {
	if row.Find("a[href]").Length() > 0 {
		return ""
	}
	text := cleanText(row.Text())
	if text == "" || !strings.HasSuffix(text, ":") || len([]rune(text)) > 60 {
		return ""
	}
	return strings.TrimSuffix(text, ":")
}

func downloadName(link *goquery.Selection, href string) string {
	if title, exists := link.Attr("title"); exists && strings.TrimSpace(title) != "" {
		return cleanText(title)
	}
	if name := cleanText(link.Text()); name != "" {
		return name
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}
	if base := path.Base(parsed.Path); base != "" && base != "/" && base != "." {
		return base
	}
	return href
}

// cellText reads a row cell by its attachment class, falling back to the
// column position used by the FLiNG table (date, size, downloads).
func cellText(cells *goquery.Selection, class string, index int) string {
	if cell := cells.Filter("." + class).First(); cell.Length() > 0 {
		return cleanText(cell.Text())
	}
	if cells.Length() > index {
		return cleanText(cells.Eq(index).Text())
	}
	return ""
}

// scanDownloadLinks is the fallback for pages without an attachments table:
// every download-looking link becomes a plain option.
func scanDownloadLinks(content *goquery.Selection, sourceURL string) []models.DownloadOption {
	var options []models.DownloadOption
	content.Find("a[href]").Each(func(_ int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || !isDownloadHref(href) {
			return
		}
		options = appendUniqueOption(options, models.DownloadOption{
			Name: downloadName(link, href),
			URL:  absoluteURL(sourceURL, href),
		})
	})
	return options
}

func appendUniqueOption(options []models.DownloadOption, candidate models.DownloadOption) []models.DownloadOption {
	for _, option := range options {
		if option.URL == candidate.URL {
			return options
		}
	}
	return append(options, candidate)
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
