package internal

// Config represents the application configuration
type Config struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	UserID   int    `json:"user_id"`
}

// AniListUserResponse represents the response from the AniList API for user info
type AniListUserResponse struct {
	Data struct {
		Viewer struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Avatar struct {
				Medium string `json:"medium"`
			} `json:"avatar"`
		} `json:"Viewer"`
	} `json:"data"`
}

// MediaListCollection represents the collection of media lists from AniList
type MediaListCollection struct {
	Data struct {
		MediaListCollection struct {
			Lists []struct {
				Name    string `json:"name"`
				Entries []struct {
					ID          int    `json:"id"`
					Status      string `json:"status"`
					Progress    int    `json:"progress"`
					Media       Media  `json:"media"`
					UpdatedAt   int    `json:"updatedAt"`
					StartedAt   Date   `json:"startedAt"`
					CompletedAt Date   `json:"completedAt"`
				} `json:"entries"`
			} `json:"lists"`
		} `json:"MediaListCollection"`
	} `json:"data"`
}

// Media represents an anime media entry from AniList
type Media struct {
	ID                int    `json:"id"`
	Title             Title  `json:"title"`
	Episodes          int    `json:"episodes"`
	Format            string `json:"format"`
	Status            string `json:"status"`
	Description       string `json:"description"`
	CoverImage        Image  `json:"coverImage"`
	MalId             int    `json:"idMal"`
	AllanimeId        string
	AverageScore      int                `json:"averageScore"`
	SeasonYear        int                `json:"seasonYear"`
	Season            string             `json:"season"`
	NextAiringEpisode *NextAiringEpisode `json:"nextAiringEpisode"`
}

type NextAiringEpisode struct {
	Episode         int `json:"episode"`
	TimeUntilAiring int `json:"timeUntilAiring"`
}

// Title represents the title of an anime
type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

// Image represents an image from AniList
type Image struct {
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

// Date represents a date from AniList
type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// AnimeEntry represents a single anime entry for display in the UI
type AnimeEntry struct {
	Title           string
	Progress        int
	Episodes        int
	ID              int
	MalId           int
	CoverImage      string
	Description     string
	NextEpisode     int
	CurrentEpisode  int
	EpisodeDuration int
	IsAiring        bool
}

// Constants for the application
var (
	LinkPriorities = []string{"filemoon", "sharepoint", "doodstream", "mp4upload"}
)
