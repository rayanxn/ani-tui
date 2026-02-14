package nyaa

import (
	"net/url"
	"strings"
	"testing"
)

func TestItemMagnetURI(t *testing.T) {
	t.Parallel()

	item := Item{Title: "Sample Show", InfoHash: "ABC123"}
	m := item.MagnetURI()
	if !strings.HasPrefix(m, "magnet:?") {
		t.Fatalf("expected magnet URI, got %q", m)
	}

	raw := strings.TrimPrefix(m, "magnet:?")
	vals, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("ParseQuery failed: %v", err)
	}
	if vals.Get("xt") != "urn:btih:ABC123" {
		t.Fatalf("unexpected xt: %q", vals.Get("xt"))
	}
	if vals.Get("dn") != "Sample Show" {
		t.Fatalf("unexpected dn: %q", vals.Get("dn"))
	}
	if len(vals["tr"]) == 0 {
		t.Fatal("expected trackers in magnet URI")
	}
}

func TestItemMagnetURI_EmptyHash(t *testing.T) {
	t.Parallel()

	if got := (Item{Title: "X"}).MagnetURI(); got != "" {
		t.Fatalf("expected empty magnet for empty hash, got %q", got)
	}
}

func TestItemIsTrusted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		trusted string
		want    bool
	}{
		{trusted: "Yes", want: true},
		{trusted: "true", want: true},
		{trusted: "1", want: true},
		{trusted: "No", want: false},
		{trusted: "", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.trusted, func(t *testing.T) {
			t.Parallel()
			if got := (Item{Trusted: tc.trusted}).IsTrusted(); got != tc.want {
				t.Fatalf("IsTrusted(%q) = %v, want %v", tc.trusted, got, tc.want)
			}
		})
	}
}
