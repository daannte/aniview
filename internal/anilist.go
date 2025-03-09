package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	aniListAPIURL = "https://graphql.anilist.co"
	timeout       = 10 * time.Second
)

// AniListClient handles communication with the AniList API
type AniListClient struct {
	httpClient *http.Client
	token      string
}

// NewAniListClient creates a new AniList client with the given token
func NewAniListClient(token string) *AniListClient {
	return &AniListClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token: token,
	}
}

// UpdateUserInfo fetches user information from AniList and updates the config
func (c *AniListClient) UpdateUserInfo(config *Config) error {
	query := `
	query {
		Viewer {
			id
			name
		}
	}
	`
	var response AniListUserResponse
	if err := c.executeQuery(query, nil, &response); err != nil {
		return fmt.Errorf("failed to fetch user info: %w", err)
	}

	config.Username = response.Data.Viewer.Name
	config.UserID = response.Data.Viewer.ID

	// Save the updated config
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}
	return nil
}

// GetPlanned fetches the user's planned anime list
func (c *AniListClient) GetPlanned(userID int) ([]AnimeEntry, error) {
	return c.getAnimeList(userID, "PLANNING")
}

// GetCurrentlyWatching fetches the user's currently watching anime list
func (c *AniListClient) GetCurrentlyWatching(userID int) ([]AnimeEntry, error) {
	return c.getAnimeList(userID, "CURRENT")
}

// getAnimeList fetches the user's anime list with the specified status
func (c *AniListClient) getAnimeList(userID int, status string) ([]AnimeEntry, error) {
	query := `
	query ($userId: Int, $status: MediaListStatus) {
		MediaListCollection(userId: $userId, type: ANIME, status: $status) {
			lists {
				name
				entries {
					id
					status
					progress
					updatedAt
					startedAt {
						year
						month
						day
					}
					completedAt {
						year
						month
						day
					}
					media {
						id
						title {
							romaji
							english
							native
						}
						episodes
						format
						status
						description
						coverImage {
							medium
							large
						}
						idMal
						averageScore
						seasonYear
						season
						nextAiringEpisode {
							episode
							timeUntilAiring
						}
					}
				}
			}
		}
	}
	`
	variables := map[string]interface{}{
		"userId": userID,
		"status": status,
	}

	var response MediaListCollection
	if err := c.executeQuery(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch anime list with status %s: %w", status, err)
	}

	return convertToAnimeEntries(response), nil
}

// UpdateProgress updates the progress of an anime
func (c *AniListClient) UpdateProgress(mediaID int, progress int) error {
	return c.UpdateAnime(mediaID, progress, "")
}

// UpdateAnime updates both progress and status of an anime
func (c *AniListClient) UpdateAnime(mediaID int, progress int, status string) error {
	query := `
	mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
		SaveMediaListEntry(mediaId: $mediaId, progress: $progress, status: $status) {
			id
			progress
			status
		}
	}
	`
	variables := map[string]interface{}{
		"mediaId":  mediaID,
		"progress": progress,
	}

	if status != "" {
		variables["status"] = status
	}

	var response map[string]interface{}
	if err := c.executeQuery(query, variables, &response); err != nil {
		return fmt.Errorf("failed to update anime (mediaID: %d): %w", mediaID, err)
	}

	return nil
}

// executeQuery executes a GraphQL query against the AniList API
func (c *AniListClient) executeQuery(query string, variables map[string]interface{}, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create the request body
	reqBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		reqBody["variables"] = variables
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", aniListAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, respBody)
	}

	// Parse the response
	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// Helper function to convert MediaListCollection to AnimeEntry slice
func convertToAnimeEntries(response MediaListCollection) []AnimeEntry {
	var animeList []AnimeEntry

	for _, list := range response.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			nextEp := entry.Progress + 1

			title := entry.Media.Title.English
			if title == "" {
				title = entry.Media.Title.Romaji
			}

			maxEpisodes := entry.Media.Episodes
			if !entry.Media.NextAiringEpisode.IsEmpty() {
				maxEpisodes = entry.Media.NextAiringEpisode.Episode - 1
			}

			animeList = append(animeList, AnimeEntry{
				Title:             title,
				Progress:          entry.Progress,
				Episodes:          maxEpisodes,
				ID:                entry.Media.ID,
				MalId:             entry.Media.MalId,
				CoverImage:        entry.Media.CoverImage.Medium,
				Description:       entry.Media.Description,
				NextEpisode:       nextEp,
				IsAiring:          !entry.Media.NextAiringEpisode.IsEmpty(),
				NextAiringEpisode: entry.Media.NextAiringEpisode,
			})
		}
	}

	return animeList
}
