package anilist

import "fmt"

const clientID = 28498

// AuthURL returns the AniList OAuth implicit grant URL.
func AuthURL() string {
	return fmt.Sprintf("https://anilist.co/api/v2/oauth/authorize?client_id=%d&response_type=token", clientID)
}
