package scraper

import (
	"strings"
	"testing"

	"changeme/backend/models"
	"github.com/PuerkitoBio/goquery"
)

const downloadSectionHTML = `
<html><body>
<div class="entry-content">
	<h2>Download</h2>
	<table class="da-attachments-table">
	<thead>
		<th class="attachment-title">File</th>
		<th class="attachment-date">Date added</th>
		<th class="attachment-size">File size</th>
		<th class="attachment-downloads">Downloads</th>
	</thead>
	<tbody>
		<tr><td colspan="4"><strong>Auto-Updating Version:</strong></td></tr>
		<tr class="zip">
			<td class="attachment-title">
				<img class="attachment-icon" src="https://flingtrainer.com/wp-content/plugins/download-attachments/images/ext/zip.gif" alt="zip" />
				<a href="https://flingtrainer.com/downloads/auto1" class="attachment-link">GAME.LatestVersion.Plus.27.Trainer-FLiNG</a>
			</td>
			<td class="attachment-date">2026-06-19 23:09</td>
			<td class="attachment-size">132 KB</td>
			<td class="attachment-downloads">105908</td>
		</tr>
		<tr><td colspan="4"><strong>Standalone Versions:</strong></td></tr>
		<tr class="zip">
			<td class="attachment-title">
				<img class="attachment-icon" src="https://flingtrainer.com/wp-content/plugins/download-attachments/images/ext/zip.gif" alt="zip" />
				<a href="/downloads/standalone1" class="attachment-link">GAME.v1.0.Plus.27.Trainer-FLiNG</a>
			</td>
			<td class="attachment-date">2026-04-17 16:51</td>
			<td class="attachment-size">13.43 MB</td>
			<td class="attachment-downloads">195185</td>
		</tr>
		<tr class="zip">
			<td class="attachment-title">
				<a href="/wp-content/uploads/2026/04/GAME.v1.0.Plus.12.Trainer-FLiNG.zip" title="GAME.v1.0.Plus.12.Trainer-FLiNG">GAME.v1.0.Plus.12.Trainer-FLiNG</a>
			</td>
			<td>2026-04-16 23:06</td>
			<td>1.02 MB</td>
			<td>31726</td>
		</tr>
	</tbody>
	</table>
</div>
</body></html>
`

func parseOptionsFromHTML(t *testing.T, html string) []models.DownloadOption {
	t.Helper()
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse fixture html: %v", err)
	}
	content := document.Find(".entry-content")
	if content.Length() == 0 {
		t.Fatalf("fixture is missing .entry-content")
	}
	return parseDownloadOptions(content, "https://flingtrainer.com/trainer/game/")
}

func TestParseDownloadOptionsParsesAllRowsAndGroups(t *testing.T) {
	options := parseOptionsFromHTML(t, downloadSectionHTML)

	if len(options) != 3 {
		t.Fatalf("expected 3 download options, got %d: %+v", len(options), options)
	}

	first := options[0]
	if first.Group != "Auto-Updating Version" {
		t.Errorf("first option group = %q, want %q", first.Group, "Auto-Updating Version")
	}
	if first.Name != "GAME.LatestVersion.Plus.27.Trainer-FLiNG" {
		t.Errorf("first option name = %q", first.Name)
	}
	if first.URL != "https://flingtrainer.com/downloads/auto1" {
		t.Errorf("first option url = %q", first.URL)
	}
	if first.DateAdded != "2026-06-19 23:09" || first.FileSize != "132 KB" || first.Downloads != "105908" {
		t.Errorf("first option metadata = %q/%q/%q", first.DateAdded, first.FileSize, first.Downloads)
	}

	second := options[1]
	if second.Group != "Standalone Versions" {
		t.Errorf("second option group = %q", second.Group)
	}
	if second.URL != "https://flingtrainer.com/downloads/standalone1" {
		t.Errorf("relative link was not absolutized: %q", second.URL)
	}

	third := options[2]
	if third.Group != "Standalone Versions" {
		t.Errorf("third option group = %q", third.Group)
	}
	if third.DateAdded != "2026-04-16 23:06" || third.FileSize != "1.02 MB" || third.Downloads != "31726" {
		t.Errorf("third option metadata via positional fallback = %q/%q/%q", third.DateAdded, third.FileSize, third.Downloads)
	}
}

func TestParseDownloadOptionsFallsBackToPlainLinks(t *testing.T) {
	html := `
	<html><body><div class="entry-content">
		<p>Download: <a href="/wp-content/uploads/2026/trainer.zip">trainer.zip</a>
		and <a href="https://example.com/other">other</a></p>
	</div></body></html>
	`
	options := parseOptionsFromHTML(t, html)

	if len(options) != 1 {
		t.Fatalf("expected 1 download option, got %d: %+v", len(options), options)
	}
	if options[0].URL != "https://flingtrainer.com/wp-content/uploads/2026/trainer.zip" {
		t.Errorf("unexpected url: %q", options[0].URL)
	}
	if options[0].Name != "trainer.zip" {
		t.Errorf("unexpected name: %q", options[0].Name)
	}
}

func TestParseDownloadOptionsDeduplicatesByURL(t *testing.T) {
	html := `
	<html><body><div class="entry-content">
		<table class="da-attachments-table"><tbody>
			<tr><td class="attachment-title"><a href="https://flingtrainer.com/downloads/same" class="attachment-link">A</a></td></tr>
			<tr><td class="attachment-title"><a href="https://flingtrainer.com/downloads/same" class="attachment-link">A</a></td></tr>
		</tbody></table>
	</div></body></html>
	`
	options := parseOptionsFromHTML(t, html)

	if len(options) != 1 {
		t.Fatalf("expected duplicate rows to collapse, got %d: %+v", len(options), options)
	}
}
