package torrent

import (
	"context"
	"fmt"

	"github.com/anacrolix/torrent"
)

const readaheadBytes = 10 * 1024 * 1024 // 10 MB

// AddMagnetAndStream adds a magnet URI, waits for metadata, and returns a
// Reader for the largest file in the torrent.
func (c *Client) AddMagnetAndStream(ctx context.Context, magnetURI string) (torrent.Reader, error) {
	if c.activeTor != nil {
		c.activeTor.Drop()
		c.activeTor = nil
	}

	t, err := c.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("add magnet: %w", err)
	}
	c.activeTor = t

	// Wait for torrent metadata (piece info, file list).
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		t.Drop()
		c.activeTor = nil
		return nil, fmt.Errorf("metadata wait cancelled: %w", ctx.Err())
	}

	// Find the largest file (the video).
	files := t.Files()
	if len(files) == 0 {
		t.Drop()
		c.activeTor = nil
		return nil, fmt.Errorf("torrent has no files")
	}

	largest := files[0]
	for _, f := range files[1:] {
		if f.Length() > largest.Length() {
			largest = f
		}
	}

	// Deprioritize all other files.
	for _, f := range files {
		if f != largest {
			f.SetPriority(torrent.PiecePriorityNone)
		}
	}

	reader := largest.NewReader()
	reader.SetReadahead(readaheadBytes)
	reader.SetResponsive()

	return reader, nil
}

// ActiveTorrent returns the currently active torrent, or nil.
func (c *Client) ActiveTorrent() *torrent.Torrent {
	return c.activeTor
}
