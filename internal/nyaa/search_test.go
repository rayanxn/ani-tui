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
