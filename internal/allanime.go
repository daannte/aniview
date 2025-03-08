package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Constants
const (
	allanimeBaseURL = "https://allanime.day"
	allanimeAPIURL  = "https://api.allanime.day/api"
	allanimeReferer = "https://allanime.to"
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"
	requestTimeout  = 10 * time.Second
	rateLimitDelay  = 50 * time.Millisecond
	socketPath      = "/tmp/iinasocket"
	iinaCliPath     = "/Applications/IINA.app/Contents/MacOS/iina-cli"
)

// RequestHeaders returns common HTTP headers for Allanime requests
func RequestHeaders() http.Header {
	headers := http.Header{}
	headers.Set("User-Agent", userAgent)
	headers.Set("Referer", allanimeReferer)
	return headers
}

// Response types
type AllanimeResponse struct {
	Data struct {
		Episode struct {
			SourceUrls []struct {
				SourceUrl string `json:"sourceUrl"`
			} `json:"sourceUrls"`
		} `json:"episode"`
	} `json:"data"`
}

type AnimeSearchResponse struct {
	Data struct {
		Shows struct {
			Edges []AnimeSearchResult `json:"edges"`
		} `json:"shows"`
	} `json:"data"`
}

type AnimeSearchResult struct {
	ID                string      `json:"_id"`
	Name              string      `json:"name"`
	EnglishName       string      `json:"englishName"`
	AvailableEpisodes interface{} `json:"availableEpisodes"`
}

// Helper types for concurrent link extraction
type episodeResult struct {
	index int
	links []string
	err   error
}

// SearchAnime searches for anime by query and returns a map of ID to anime name
func SearchAnime(query, mode string) (map[string]string, error) {
	animeList := make(map[string]string)

	searchGql := `query($search: SearchInput, $limit: Int, $page: Int, $translationType: VaildTranslationTypeEnumType, $countryOrigin: VaildCountryOriginEnumType) {
		shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin) {
			edges {
				_id
				name
				englishName
				availableEpisodes
				__typename
			}
		}
	}`

	// Prepare the GraphQL variables
	variables := map[string]interface{}{
		"search": map[string]interface{}{
			"allowAdult":   false,
			"allowUnknown": false,
			"query":        query,
		},
		"limit":           40,
		"page":            1,
		"translationType": mode,
		"countryOrigin":   "ALL",
	}

	// Marshal the variables to JSON
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return animeList, fmt.Errorf("error marshaling variables: %w", err)
	}

	// Build the request URL
	requestURL := fmt.Sprintf("%s?variables=%s&query=%s",
		allanimeAPIURL,
		url.QueryEscape(string(variablesJSON)),
		url.QueryEscape(searchGql))

	// Make the HTTP request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return animeList, fmt.Errorf("error creating request: %w", err)
	}

	for key, value := range RequestHeaders() {
		req.Header[key] = value
	}

	// Send request
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return animeList, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return animeList, fmt.Errorf("error reading response: %w", err)
	}

	var response AnimeSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return animeList, fmt.Errorf("error parsing response: %w", err)
	}

	// Process results
	for _, anime := range response.Data.Shows.Edges {
		var episodesStr string
		if episodes, ok := anime.AvailableEpisodes.(map[string]interface{}); ok {
			if subEpisodes, ok := episodes["sub"].(float64); ok {
				episodesStr = fmt.Sprintf("%d", int(subEpisodes))
			} else {
				episodesStr = "Unknown"
			}
		}
		displayName := anime.EnglishName
		animeList[anime.ID] = fmt.Sprintf("%s (%s episodes)", displayName, episodesStr)
	}

	return animeList, nil
}

// GetEpisodeURL gets stream URLs for a specific episode of an anime
func GetEpisodeURL(id string, epNo int) ([]string, error) {
	// Prepare GraphQL query
	query := `query($showId:String!,$translationType:VaildTranslationTypeEnumType!,$episodeString:String!){episode(showId:$showId,translationType:$translationType,episodeString:$episodeString){episodeString sourceUrls}}`
	variables := map[string]string{
		"showId":          id,
		"translationType": "sub",
		"episodeString":   fmt.Sprintf("%d", epNo),
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error marshaling variables: %w", err)
	}

	// Build request URL
	values := url.Values{}
	values.Set("query", query)
	values.Set("variables", string(variablesJSON))
	reqURL := fmt.Sprintf("%s?%s", allanimeAPIURL, values.Encode())

	// Send request
	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	for key, value := range RequestHeaders() {
		req.Header[key] = value
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var response AllanimeResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Process source URLs
	return processSourceURLs(response.Data.Episode.SourceUrls)
}

// processSourceURLs processes the source URLs from the API response
func processSourceURLs(sourceUrls []struct {
	SourceUrl string `json:"sourceUrl"`
},
) ([]string, error) {
	// Pre-count valid URLs and create slice to preserve order
	validURLs := make([]string, 0)
	highestPriority := -1
	var highestPriorityURL string

	// First pass - collect valid URLs and find highest priority one
	for _, url := range sourceUrls {
		if len(url.SourceUrl) > 2 && unicode.IsDigit(rune(url.SourceUrl[2])) {
			decodedURL := decodeProviderID(url.SourceUrl[2:])

			// Check if it contains a high priority domain
			if strings.Contains(decodedURL, LinkPriorities[0]) {
				priority := int(url.SourceUrl[2] - '0')
				if priority > highestPriority {
					highestPriority = priority
					highestPriorityURL = url.SourceUrl
				}
			} else {
				validURLs = append(validURLs, url.SourceUrl)
			}
		}
	}

	// If we found a highest priority URL, use only that
	if highestPriorityURL != "" {
		validURLs = []string{highestPriorityURL}
	}

	if len(validURLs) == 0 {
		return nil, fmt.Errorf("no valid source URLs found in response")
	}

	return extractVideoLinks(validURLs)
}

// extractVideoLinks extracts video links from the provider URLs concurrently
func extractVideoLinks(validURLs []string) ([]string, error) {
	// Create channels for results
	results := make(chan episodeResult, len(validURLs))
	orderedResults := make([][]string, len(validURLs))
	highPriorityLink := make(chan []string, 1)

	// Create rate limiter to avoid overloading the server
	rateLimiter := time.NewTicker(rateLimitDelay)
	defer rateLimiter.Stop()

	// Launch goroutines to process each URL concurrently
	for i, sourceUrl := range validURLs {
		go processProviderURL(i, sourceUrl, rateLimiter.C, results, highPriorityLink)
	}

	// First, try to get a high priority link with a short timeout
	select {
	case links := <-highPriorityLink:
		// Continue extracting other links in background
		go collectRemainingResults(results, orderedResults, len(validURLs))
		return links, nil
	case <-time.After(2 * time.Second): // Short wait for high priority link
		// No high priority link found quickly, proceed with normal collection
	}

	// Collect results with timeout
	timeout := time.After(requestTimeout)
	var collectedErrors []error
	successCount := 0

	// Collect results maintaining order
	for successCount < len(validURLs) {
		select {
		case res := <-results:
			if res.err != nil {
				collectedErrors = append(collectedErrors, fmt.Errorf("URL %d: %w", res.index+1, res.err))
			} else {
				orderedResults[res.index] = res.links
				successCount++
			}
		case <-timeout:
			if successCount > 0 {
				// Flatten available results
				return flattenResults(orderedResults), nil
			}
			return nil, fmt.Errorf("timeout waiting for results after %d successful responses", successCount)
		}
	}

	// Flatten and return results
	allLinks := flattenResults(orderedResults)
	if len(allLinks) == 0 {
		return nil, fmt.Errorf("no valid links found from %d URLs: %v", len(validURLs), collectedErrors)
	}

	return allLinks, nil
}

// processProviderURL processes a single provider URL and extracts video links
func processProviderURL(idx int, url string, rateLimiterC <-chan time.Time, results chan<- episodeResult, highPriorityLink chan<- []string) {
	<-rateLimiterC // Rate limit the requests

	// Decode the provider ID
	decodedProviderID := decodeProviderID(url[2:])

	// Extract links
	extractedLinks := extractLinks(decodedProviderID)
	if extractedLinks == nil {
		results <- episodeResult{
			index: idx,
			err:   fmt.Errorf("failed to extract links for provider %s", decodedProviderID),
		}
		return
	}

	// Process links from response
	linksInterface, ok := extractedLinks["links"].([]interface{})
	if !ok {
		results <- episodeResult{
			index: idx,
			err:   fmt.Errorf("links field is not []interface{} for provider %s", decodedProviderID),
		}
		return
	}

	// Extract links from response
	var links []string
	for _, linkInterface := range linksInterface {
		linkMap, ok := linkInterface.(map[string]interface{})
		if !ok {
			continue
		}

		link, ok := linkMap["link"].(string)
		if !ok {
			continue
		}

		links = append(links, link)
	}

	// Check if any links are high priority
	for _, link := range links {
		for _, domain := range LinkPriorities[:3] { // Check only top 3 priority domains
			if strings.Contains(link, domain) {
				// Found high priority link, send it immediately
				select {
				case highPriorityLink <- []string{link}:
				default:
					// Channel already has a high priority link
				}
				break
			}
		}
	}

	results <- episodeResult{
		index: idx,
		links: links,
	}
}

// decodeProviderID decodes the encrypted provider ID
func decodeProviderID(encoded string) string {
	// Split the string into pairs of characters
	re := regexp.MustCompile("..")
	pairs := re.FindAllString(encoded, -1)

	// Mapping for the replacements
	replacements := map[string]string{
		"01": "9", "08": "0", "05": "=", "0a": "2", "0b": "3", "0c": "4", "07": "?",
		"00": "8", "5c": "d", "0f": "7", "5e": "f", "17": "/", "54": "l", "09": "1",
		"48": "p", "4f": "w", "0e": "6", "5b": "c", "5d": "e", "0d": "5", "53": "k",
		"1e": "&", "5a": "b", "59": "a", "4a": "r", "4c": "t", "4e": "v", "57": "o",
		"51": "i",
	}

	// Perform the replacement
	for i, pair := range pairs {
		if val, exists := replacements[pair]; exists {
			pairs[i] = val
		}
	}

	// Join the modified pairs back into a single string
	result := strings.Join(pairs, "")

	// Replace "/clock" with "/clock.json"
	result = strings.ReplaceAll(result, "/clock", "/clock.json")

	return result
}

// extractLinks retrieves video data from the provider URL
func extractLinks(providerID string) map[string]interface{} {
	url := allanimeBaseURL + providerID

	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("GET", url, nil)

	var videoData map[string]interface{}
	if err != nil {
		return videoData
	}

	// Add headers
	for key, value := range RequestHeaders() {
		req.Header[key] = value
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return videoData
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return videoData
	}

	// Parse JSON
	err = json.Unmarshal(body, &videoData)
	if err != nil {
		return videoData
	}

	return videoData
}

// collectRemainingResults collects remaining results in background
func collectRemainingResults(results chan episodeResult, orderedResults [][]string, remainingURLs int) {
	successCount := 0

	for successCount < remainingURLs {
		select {
		case res := <-results:
			if res.err == nil {
				orderedResults[res.index] = res.links
				successCount++
			}
		case <-time.After(requestTimeout):
			return
		}
	}
}

// flattenResults converts the ordered slice of link slices into a single slice
func flattenResults(results [][]string) []string {
	var totalLen int
	for _, r := range results {
		totalLen += len(r)
	}

	allLinks := make([]string, 0, totalLen)
	for _, links := range results {
		allLinks = append(allLinks, links...)
	}

	return allLinks
}

// PrioritizeLink selects the best link from available options
func PrioritizeLink(links []string) string {
	if len(links) == 0 {
		return ""
	}

	// First priority: SharePoint links
	// for _, link := range links {
	// 	if strings.Contains(link, "sharepoint") {
	// 		return link
	// 	}
	// }

	// Create a map for quick lookup of priorities
	priorityMap := make(map[string]int)
	for i, p := range LinkPriorities {
		priorityMap[p] = len(LinkPriorities) - i // Higher index means higher priority
	}

	// Find link with highest priority
	highestPriority := -1
	var bestLink string

	for _, link := range links {
		for domain, priority := range priorityMap {
			if strings.Contains(link, domain) {
				if priority > highestPriority {
					highestPriority = priority
					bestLink = link
				}
				break
			}
		}
	}

	// If no priority link found, return the first link
	if bestLink == "" {
		return links[0]
	}

	return bestLink
}

// PlayEpisode plays an episode using IINA with a custom title
func PlayEpisode(links []string, anime AnimeEntry) error {
	if len(links) == 0 {
		return fmt.Errorf("no links available to play")
	}

	// Choose the best link
	link := PrioritizeLink(links)

	// Ensure that iina-cli is available
	_, err := exec.LookPath(iinaCliPath)
	if err != nil {
		return fmt.Errorf("iina-cli not found: %w", err)
	}

	// Prepare arguments for IINA
	cmdArgs := []string{
		"--no-stdin",
		"--keep-running",
		"--mpv-force-media-title=" + anime.Title,
		"--mpv-input-ipc-server=" + socketPath,
		link,
	}

	cmd := exec.Command(iinaCliPath, cmdArgs...)

	// Set up Discord presence updating
	quit := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Wait for the socket file to exist before proceeding
		if err := waitForSocket(socketPath); err != nil {
			fmt.Printf("Error waiting for socket: %v\n", err)
			return
		}

		// Update Discord presence while video is playing
		for {
			select {
			case <-quit:
				return
			default:
				if err := DiscordPresence("1285024019447287921", anime); err != nil {
					fmt.Printf("Error updating Discord presence: %v\n", err)
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Run IINA
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run IINA: %w", err)
	}

	// Clean up
	close(quit)
	wg.Wait()

	return nil
}
