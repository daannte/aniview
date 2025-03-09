package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	aniListAPIURL = "https://graphql.anilist.co"
)

// UpdateUserInfo fetches user information from AniList and updates the config
func UpdateUserInfo(config *Config) error {
	query := `
	query {
		Viewer {
			id
			name
		}
	}
	`

	var response AniListUserResponse
	if err := executeAniListQuery(config.Token, query, nil, &response); err != nil {
		return fmt.Errorf("failed to fetch user info: %v", err)
	}

	config.Username = response.Data.Viewer.Name
	config.UserID = response.Data.Viewer.ID

	// Save the updated config
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save updated config: %v", err)
	}

	return nil
}

// Fetches the user's planned anime list
func GetPlanned(config *Config) ([]AnimeEntry, error) {
	query := `
	query ($userId: Int) {
		MediaListCollection(userId: $userId, type: ANIME, status: PLANNING) {
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
		"userId": config.UserID,
	}

	var response MediaListCollection
	if err := executeAniListQuery(config.Token, query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch watching list: %v", err)
	}

	// Convert the response to a simpler format for the UI
	var animeList []AnimeEntry
	for _, list := range response.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			nextEp := entry.Progress + 1

			title := entry.Media.Title.English
			if title == "" {
				title = entry.Media.Title.Romaji
			}

			maxEpisodes := entry.Media.Episodes
			if entry.Media.NextAiringEpisode != nil {
				maxEpisodes = entry.Media.NextAiringEpisode.Episode - 1
			}

			animeList = append(animeList, AnimeEntry{
				Title:       title,
				Progress:    entry.Progress,
				Episodes:    maxEpisodes,
				ID:          entry.Media.ID,
				MalId:       entry.Media.MalId,
				CoverImage:  entry.Media.CoverImage.Medium,
				Description: entry.Media.Description,
				NextEpisode: nextEp,
				IsAiring:    entry.Media.NextAiringEpisode != nil,
			})
		}
	}

	return animeList, nil
}

// Fetches the user's currently watching anime list
func GetCurrentlyWatching(config *Config) ([]AnimeEntry, error) {
	query := `
	query ($userId: Int) {
		MediaListCollection(userId: $userId, type: ANIME, status: CURRENT) {
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
		"userId": config.UserID,
	}

	var response MediaListCollection
	if err := executeAniListQuery(config.Token, query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch watching list: %v", err)
	}

	// Convert the response to a simpler format for the UI
	var animeList []AnimeEntry
	for _, list := range response.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			nextEp := entry.Progress + 1

			title := entry.Media.Title.English
			if title == "" {
				title = entry.Media.Title.Romaji
			}

			maxEpisodes := entry.Media.Episodes
			if entry.Media.NextAiringEpisode != nil {
				maxEpisodes = entry.Media.NextAiringEpisode.Episode - 1
			}

			animeList = append(animeList, AnimeEntry{
				Title:       title,
				Progress:    entry.Progress,
				Episodes:    maxEpisodes,
				ID:          entry.Media.ID,
				MalId:       entry.Media.MalId,
				CoverImage:  entry.Media.CoverImage.Medium,
				Description: entry.Media.Description,
				NextEpisode: nextEp,
				IsAiring:    entry.Media.NextAiringEpisode != nil,
			})
		}
	}

	return animeList, nil
}

// UpdateProgress updates the progress of an anime
func UpdateProgress(config *Config, mediaID int, progress int) error {
	query := `
	mutation ($mediaId: Int, $progress: Int) {
		SaveMediaListEntry(mediaId: $mediaId, progress: $progress) {
			id
			progress
		}
	}
	`

	variables := map[string]interface{}{
		"mediaId":  mediaID,
		"progress": progress,
	}

	var response map[string]interface{}
	if err := executeAniListQuery(config.Token, query, variables, &response); err != nil {
		return fmt.Errorf("failed to update progress: %v", err)
	}

	return nil
}

func UpdateAnime(config *Config, mediaID int, progress int, status string) error {
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
		"status":   status,
	}

	var response map[string]interface{}
	if err := executeAniListQuery(config.Token, query, variables, &response); err != nil {
		return fmt.Errorf("failed to update anime: %v", err)
	}

	return nil
}

// executeAniListQuery executes a GraphQL query against the AniList API
func executeAniListQuery(token string, query string, variables map[string]interface{}, result interface{}) error {
	// Create the request body
	reqBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		reqBody["variables"] = variables
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", aniListAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read and parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, respBody)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	return nil
}
