package anilist

// Title represents anime title in multiple languages
type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

// DisplayTitle returns the preferred title, falling back to Romaji if English is not available
func (t Title) DisplayTitle() string {
	if t.English != "" {
		return t.English
	}
	return t.Romaji
}

// Studio represents an animation studio
type Studio struct {
	Name string `json:"name"`
}

// StudioConnection wraps studio nodes
type StudioConnection struct {
	Nodes []Studio `json:"nodes"`
}

// AiringSchedule represents next airing episode info
type AiringSchedule struct {
	Episode         int   `json:"episode"`
	AiringAt        int64 `json:"airingAt"`
	TimeUntilAiring int   `json:"timeUntilAiring"`
}

// PageInfo holds pagination data from AniList's Page queries
type PageInfo struct {
	Total       int  `json:"total"`
	CurrentPage int  `json:"currentPage"`
	LastPage    int  `json:"lastPage"`
	HasNextPage bool `json:"hasNextPage"`
}

// Media represents an anime
type Media struct {
	ID                 int               `json:"id"`
	Title              Title             `json:"title"`
	Description        string            `json:"description"`
	Episodes           int               `json:"episodes"`
	Duration           int               `json:"duration"` // duration in minutes per episode
	Status             string            `json:"status"`   // FINISHED, RELEASING, NOT_YET_RELEASED, etc.
	Format             string            `json:"format"`   // TV, MOVIE, OVA, ONA, etc.
	Genres             []string          `json:"genres"`
	AverageScore       int               `json:"averageScore"`
	Studios            StudioConnection  `json:"studios"`
	NextAiringEpisode  *AiringSchedule   `json:"nextAiringEpisode"`
}

// MediaList represents a user's anime list entry
type MediaList struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`  // CURRENT, PLANNING, COMPLETED, DROPPED, PAUSED
	Progress  int    `json:"progress"` // number of episodes watched
	Score     int    `json:"score"`
	Media     Media  `json:"media"`
}

// MediaListGroup represents a group of media list entries by status
type MediaListGroup struct {
	Status  string      `json:"status"`
	Entries []MediaList `json:"entries"`
}

// MediaListCollection wraps list groups
type MediaListCollection struct {
	Lists []MediaListGroup `json:"lists"`
}

// User represents an AniList user
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
