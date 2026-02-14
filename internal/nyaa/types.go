package nyaa

import (
	"fmt"
	"net/url"
	"strings"
)

// RSS is the root structure for the nyaa.si RSS feed.
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel contains RSS feed metadata and items.
type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

// Item represents a single torrent result from nyaa.si RSS.
type Item struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	GUID      string `xml:"guid"`
	PubDate   string `xml:"pubDate"`
	Category  string `xml:"category"`
	Size      string `xml:"size"`
	Downloads int    `xml:"https://nyaa.si/xmlns/nyaa downloads"`
	InfoHash  string `xml:"https://nyaa.si/xmlns/nyaa infoHash"`
	Seeders   int    `xml:"https://nyaa.si/xmlns/nyaa seeders"`
	Leechers  int    `xml:"https://nyaa.si/xmlns/nyaa leechers"`
	Trusted   string `xml:"https://nyaa.si/xmlns/nyaa trusted"`
	Remake    string `xml:"https://nyaa.si/xmlns/nyaa remake"`
}

// MagnetURI builds a magnet link from the item's info hash and title.
func (i Item) MagnetURI() string {
	hash := strings.TrimSpace(i.InfoHash)
	if hash == "" {
		return ""
	}

	displayName := strings.TrimSpace(i.Title)
	if displayName == "" {
		displayName = hash
	}

	trackers := []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.demonii.com:1337/announce",
		"udp://tracker.torrent.eu.org:451/announce",
		"udp://tracker-udp.gbitt.info:80/announce",
		"udp://exodus.desync.com:6969/announce",
	}

	parts := []string{
		"magnet:?xt=urn:btih:" + url.QueryEscape(hash),
		"dn=" + url.QueryEscape(displayName),
	}
	for _, tr := range trackers {
		parts = append(parts, "tr="+url.QueryEscape(tr))
	}

	return strings.Join(parts, "&")
}

// IsTrusted reports whether the torrent is marked trusted by nyaa.si.
func (i Item) IsTrusted() bool {
	switch strings.ToLower(strings.TrimSpace(i.Trusted)) {
	case "yes", "true", "1":
		return true
	default:
		return false
	}
}

// Summary returns a compact line with key torrent metadata.
func (i Item) Summary() string {
	trusted := "No"
	if i.IsTrusted() {
		trusted = "Yes"
	}
	return fmt.Sprintf("S:%d | L:%d | %s | Trusted: %s", i.Seeders, i.Leechers, i.Size, trusted)
}
