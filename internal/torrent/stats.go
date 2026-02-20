package torrent

import (
	"fmt"

	"github.com/anacrolix/torrent"
)

// Stats holds a snapshot of torrent download progress.
type Stats struct {
	BytesCompleted int64
	BytesTotal     int64
	Peers          int
	Seeders        int
}

// GetStats returns a snapshot of the torrent's current download stats.
func GetStats(t *torrent.Torrent) Stats {
	stats := t.Stats()
	info := t.Info()

	var total, completed int64
	if info != nil {
		total = info.TotalLength()
		completed = total - t.BytesMissing()
	}

	return Stats{
		BytesCompleted: completed,
		BytesTotal:     total,
		Peers:          stats.ActivePeers,
		Seeders:        stats.ConnectedSeeders,
	}
}

// Progress returns the download progress as a float between 0.0 and 1.0.
func (s Stats) Progress() float64 {
	if s.BytesTotal == 0 {
		return 0
	}
	return float64(s.BytesCompleted) / float64(s.BytesTotal)
}

// FormatBytes formats a byte count into a human-readable string.
func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatSpeed formats a bytes-per-second value into a human-readable string.
func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}
