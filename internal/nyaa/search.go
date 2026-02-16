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
)

const defaultFeedURL = "https://nyaa.si/"

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
	q.Set("c", "1_2")
	q.Set("f", "0")
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
