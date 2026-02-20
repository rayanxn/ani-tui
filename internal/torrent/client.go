package torrent

import (
	"fmt"
	"os"

	"github.com/anacrolix/torrent"
)

// Client wraps an anacrolix/torrent client for streaming.
type Client struct {
	client      *torrent.Client
	activeTor   *torrent.Torrent
	downloadDir string
	ownsTempDir bool
}

// NewClient creates a torrent client configured for streaming (no seeding).
func NewClient(downloadDir string) (*Client, error) {
	ownsTempDir := false
	if downloadDir == "" {
		dir, err := os.MkdirTemp("", "ani-tui-*")
		if err != nil {
			return nil, fmt.Errorf("create temp download dir: %w", err)
		}
		downloadDir = dir
		ownsTempDir = true
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = downloadDir
	cfg.Seed = false

	tc, err := torrent.NewClient(cfg)
	if err != nil {
		if ownsTempDir {
			os.RemoveAll(downloadDir)
		}
		return nil, fmt.Errorf("create torrent client: %w", err)
	}

	return &Client{
		client:      tc,
		downloadDir: downloadDir,
		ownsTempDir: ownsTempDir,
	}, nil
}

// Close drops the active torrent and closes the underlying client.
func (c *Client) Close() {
	if c.activeTor != nil {
		c.activeTor.Drop()
		c.activeTor = nil
	}
	if c.client != nil {
		c.client.Close()
	}
	if c.ownsTempDir {
		os.RemoveAll(c.downloadDir)
	}
}
