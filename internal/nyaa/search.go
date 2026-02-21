package nyaa

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

const defaultFeedURL = "https://nyaa.si/"

const (
	// categoryAnimeEnglish filters results to Anime > English-translated (c=1_2).
	categoryAnimeEnglish = "1_2"
	// filterNoFilter applies no torrent filter (f=0: show all results).
	filterNoFilter = "0"
)

// Client fetches and decodes nyaa RSS results.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Client with sane defaults.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		BaseURL:    defaultFeedURL,
		HTTPClient: httpClient,
	}
}

var defaultClient = NewClient(nil)

// Search fetches nyaa.si RSS results for a query and sorts by seeders desc.
func Search(ctx context.Context, query string) ([]Item, error) {
	return defaultClient.Search(ctx, query)
}

// Search fetches nyaa.si RSS results for a query and sorts by seeders desc.
func (c *Client) Search(ctx context.Context, query string) ([]Item, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty nyaa query")
	}

	baseURL := c.BaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultFeedURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse nyaa url: %w", err)
	}

	q := u.Query()
	q.Set("page", "rss")
	q.Set("q", query)
	q.Set("c", categoryAnimeEnglish)
	q.Set("f", filterNoFilter)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nyaa request failed: status %d", resp.StatusCode)
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("decode rss: %w", err)
	}

	items := rss.Channel.Items
	sort.Slice(items, func(i, j int) bool {
		if items[i].Seeders == items[j].Seeders {
			return items[i].Downloads > items[j].Downloads
		}
		return items[i].Seeders > items[j].Seeders
	})

	return items, nil
}

// SearchRequest bundles all parameters needed for a torrent search.
type SearchRequest struct {
	PrimaryTitle string
	AltTitles    []string
	Episode      int
	Quality      string
}

// SearchWithFallback searches the primary title first, filters results, and
// lazily tries non-CJK alt titles if the filtered count is below minResults.
// Results are deduplicated by InfoHash and re-sorted.
func (c *Client) SearchWithFallback(ctx context.Context, req SearchRequest) ([]Item, error) {
	const minResults = 3

	primaryQuery := BuildSearchQuery(req.PrimaryTitle, req.Episode, req.Quality)
	items, err := c.Search(ctx, primaryQuery)
	if err != nil {
		return nil, err
	}

	filtered := FilterByTitle(items, req.AltTitles)
	if len(filtered) >= minResults {
		return filtered, nil
	}

	// Collect already-seen info hashes from primary results.
	seen := make(map[string]bool)
	for _, it := range items {
		if it.InfoHash != "" {
			seen[it.InfoHash] = true
		}
	}

	// Try each non-CJK alt title that produces a different query.
	for _, alt := range req.AltTitles {
		if IsLikelyCJK(alt) {
			continue
		}
		q := BuildSearchQuery(alt, req.Episode, req.Quality)
		if q == primaryQuery {
			continue
		}

		extra, searchErr := c.Search(ctx, q)
		if searchErr != nil {
			continue
		}
		for _, it := range extra {
			if it.InfoHash != "" && seen[it.InfoHash] {
				continue
			}
			if it.InfoHash != "" {
				seen[it.InfoHash] = true
			}
			items = append(items, it)
		}
	}

	// Re-filter and re-sort the merged set.
	filtered = FilterByTitle(items, req.AltTitles)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Seeders == filtered[j].Seeders {
			return filtered[i].Downloads > filtered[j].Downloads
		}
		return filtered[i].Seeders > filtered[j].Seeders
	})
	return filtered, nil
}

// SearchWithFallback uses the default client.
func SearchWithFallback(ctx context.Context, req SearchRequest) ([]Item, error) {
	return defaultClient.SearchWithFallback(ctx, req)
}

// IsLikelyCJK returns true if the string contains CJK (Han, Katakana, Hiragana) characters.
// Used to skip native Japanese/Chinese titles from Nyaa search queries.
func IsLikelyCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r) {
			return true
		}
	}
	return false
}

// BuildSearchQuery builds a nyaa query like: "Title 02 1080p".
func BuildSearchQuery(title string, episode int, quality string) string {
	parts := make([]string, 0, 3)
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle != "" {
		parts = append(parts, cleanTitle)
	}
	if episode > 0 {
		parts = append(parts, fmt.Sprintf("%02d", episode))
	}
	if q := strings.TrimSpace(quality); q != "" {
		parts = append(parts, q)
	}
	return strings.Join(parts, " ")
}
