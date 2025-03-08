package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetEpisodeData fetches episode data for a given anime ID and episode number
func GetEpisodeData(animeID int, episodeNo int, anime *AnimeEntry) error {
	url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d/episodes/%d", animeID, episodeNo)

	// Use the helper function for making the GET request
	response, err := makeGetRequest(url, nil)
	if err != nil {
		return nil
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil
	}
	// getStringValue := func(field string) string {
	// 	if value, ok := data[field].(string); ok {
	// 		return value
	// 	}
	// 	return ""
	// }

	getIntValue := func(field string) int {
		if value, ok := data[field].(float64); ok {
			return int(value)
		}
		return 0
	}

	// getBoolValue := func(field string) bool {
	// 	if value, ok := data[field].(bool); ok {
	// 		return value
	// 	}
	// 	return false
	// }

	anime.EpisodeDuration = getIntValue("duration")

	return nil
}

func makeGetRequest(url string, headers map[string]string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with status %d: %s", resp.StatusCode, body)
	}

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return responseData, nil
}
