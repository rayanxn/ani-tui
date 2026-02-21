package nyaa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSearch_DecodesAndSortsBySeedersThenDownloads(t *testing.T) {
	t.Parallel()

	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = map[string]string{
			"page": r.URL.Query().Get("page"),
			"q":    r.URL.Query().Get("q"),
			"c":    r.URL.Query().Get("c"),
			"f":    r.URL.Query().Get("f"),
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel>
    <title>Nyaa</title>
    <item>
      <title>B</title>
      <size>500 MiB</size>
      <nyaa:downloads>10</nyaa:downloads>
      <nyaa:infoHash>bbb</nyaa:infoHash>
      <nyaa:seeders>20</nyaa:seeders>
      <nyaa:leechers>4</nyaa:leechers>
      <nyaa:trusted>Yes</nyaa:trusted>
    </item>
    <item>
      <title>A</title>
      <size>700 MiB</size>
      <nyaa:downloads>50</nyaa:downloads>
      <nyaa:infoHash>aaa</nyaa:infoHash>
      <nyaa:seeders>20</nyaa:seeders>
      <nyaa:leechers>2</nyaa:leechers>
      <nyaa:trusted>No</nyaa:trusted>
    </item>
    <item>
      <title>C</title>
      <size>400 MiB</size>
      <nyaa:downloads>100</nyaa:downloads>
      <nyaa:infoHash>ccc</nyaa:infoHash>
      <nyaa:seeders>10</nyaa:seeders>
      <nyaa:leechers>1</nyaa:leechers>
      <nyaa:trusted>No</nyaa:trusted>
    </item>
  </channel>
</rss>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	items, err := client.Search(context.Background(), "one piece")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if gotQuery["page"] != "rss" || gotQuery["q"] != "one piece" || gotQuery["c"] != "1_2" || gotQuery["f"] != "0" {
		t.Fatalf("unexpected query params: %#v", gotQuery)
	}

	if items[0].Title != "A" || items[1].Title != "B" || items[2].Title != "C" {
		t.Fatalf("unexpected sort order: %q, %q, %q", items[0].Title, items[1].Title, items[2].Title)
	}
}

func TestClientSearch_ErrorsOnNon200(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	_, err := client.Search(context.Background(), "naruto")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClientSearch_ErrorsOnEmptyQuery(t *testing.T) {
	t.Parallel()

	client := NewClient(nil)
	_, err := client.Search(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchWithFallback_PrimaryEnough(t *testing.T) {
	t.Parallel()

	// Return 3+ matching items so fallback is not triggered.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel>
    <title>Nyaa</title>
    <item><title>[Sub] My Anime - 01</title><size>500 MiB</size><nyaa:downloads>10</nyaa:downloads><nyaa:infoHash>aaa</nyaa:infoHash><nyaa:seeders>50</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
    <item><title>[Sub] My Anime - 01</title><size>400 MiB</size><nyaa:downloads>5</nyaa:downloads><nyaa:infoHash>bbb</nyaa:infoHash><nyaa:seeders>30</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
    <item><title>[Sub] My Anime - 01</title><size>300 MiB</size><nyaa:downloads>3</nyaa:downloads><nyaa:infoHash>ccc</nyaa:infoHash><nyaa:seeders>10</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
  </channel>
</rss>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	items, err := client.SearchWithFallback(context.Background(), SearchRequest{
		PrimaryTitle: "My Anime",
		AltTitles:    []string{"My Anime", "Alt Title"},
		Episode:      1,
		Quality:      "1080p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestSearchWithFallback_FallsBackOnFewResults(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/rss+xml")
		if q == "My Anime 01 1080p" {
			// Primary returns 1 matching item (below threshold of 3).
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel><title>Nyaa</title>
    <item><title>[Sub] My Anime - 01</title><size>500 MiB</size><nyaa:downloads>10</nyaa:downloads><nyaa:infoHash>aaa</nyaa:infoHash><nyaa:seeders>50</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
  </channel>
</rss>`))
		} else {
			// Fallback returns an extra item.
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel><title>Nyaa</title>
    <item><title>[Sub] My Anime - 01</title><size>400 MiB</size><nyaa:downloads>5</nyaa:downloads><nyaa:infoHash>bbb</nyaa:infoHash><nyaa:seeders>30</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
  </channel>
</rss>`))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	items, err := client.SearchWithFallback(context.Background(), SearchRequest{
		PrimaryTitle: "My Anime",
		AltTitles:    []string{"My Anime", "Alt Name"},
		Episode:      1,
		Quality:      "1080p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items after fallback, got %d", len(items))
	}
	if callCount < 2 {
		t.Fatalf("expected fallback search, but only %d calls made", callCount)
	}
}

func TestSearchWithFallback_SkipsCJKAltTitles(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel><title>Nyaa</title></channel>
</rss>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	_, _ = client.SearchWithFallback(context.Background(), SearchRequest{
		PrimaryTitle: "My Anime",
		AltTitles:    []string{"My Anime", "マイアニメ"},
		Episode:      1,
	})
	// Only primary query should fire; CJK alt title skipped, "My Anime" produces same query as primary.
	if callCount != 1 {
		t.Fatalf("expected 1 call (CJK skipped, same-query skipped), got %d", callCount)
	}
}

func TestSearchWithFallback_DeduplicatesByInfoHash(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		// Both queries return the same item.
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:nyaa="https://nyaa.si/xmlns/nyaa">
  <channel><title>Nyaa</title>
    <item><title>[Sub] My Anime - 01</title><size>500 MiB</size><nyaa:downloads>10</nyaa:downloads><nyaa:infoHash>aaa</nyaa:infoHash><nyaa:seeders>50</nyaa:seeders><nyaa:leechers>1</nyaa:leechers><nyaa:trusted>No</nyaa:trusted></item>
  </channel>
</rss>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	client.BaseURL = srv.URL

	items, err := client.SearchWithFallback(context.Background(), SearchRequest{
		PrimaryTitle: "My Anime",
		AltTitles:    []string{"My Anime", "Alt Name"},
		Episode:      1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deduplicated item, got %d", len(items))
	}
}

func TestIsLikelyCJK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"Oshi no Ko", false},
		{"推しの子", true},
		{"おしのこ", true},
		{"オシノコ", true},
		{"Attack on Titan", false},
		{"進撃の巨人", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := IsLikelyCJK(tt.input); got != tt.want {
				t.Errorf("IsLikelyCJK(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	t.Parallel()

	if got := BuildSearchQuery("One Piece", 2, "1080p"); got != "One Piece 02 1080p" {
		t.Fatalf("unexpected query: %q", got)
	}
	if got := BuildSearchQuery("One Piece", 0, ""); got != "One Piece" {
		t.Fatalf("unexpected query: %q", got)
	}
	if got := BuildSearchQuery("", 12, "720p"); got != "12 720p" {
		t.Fatalf("unexpected query: %q", got)
	}
}
