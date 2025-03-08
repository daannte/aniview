package internal

import (
	"fmt"

	"github.com/daannte/rich-go/client"
)

func DiscordPresence(clientId string, anime AnimeEntry) error {
	err := client.Login(clientId)
	if err != nil {
		return err
	}

	timePos, err := getCurrentPlaybackTime("/tmp/iinasocket")
	if err != nil {
		return err
	}

	isPaused, err := getPausedState("/tmp/iinasocket")
	if err != nil {
		return err
	}

	var state string
	if isPaused {
		state = fmt.Sprintf("\nEpisode %d - %s (Paused)", anime.CurrentEpisode, FormatTime(int(timePos)))
	} else {
		if anime.EpisodeDuration != 0 {
			state = fmt.Sprintf("\nEpisode %d - %s / %s", anime.CurrentEpisode, FormatTime(int(timePos)), FormatTime(anime.EpisodeDuration))
		} else {
			state = fmt.Sprintf("\nEpisode %d - %s", anime.CurrentEpisode, FormatTime(int(timePos)))
		}
	}

	err = client.SetActivity(client.Activity{
		Type:       3,
		Details:    anime.Title,
		LargeImage: anime.CoverImage,
		LargeText:  anime.Title,
		State:      state,
		Buttons: []*client.Button{
			{
				Label: "View on AniList",
				Url:   fmt.Sprintf("https://anilist.co/anime/%d", anime.ID),
			},
			{
				Label: "View on MAL",
				Url:   fmt.Sprintf("https://myanimelist.net/anime/%d", anime.MalId),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func FormatTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	remainingSeconds := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, remainingSeconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, remainingSeconds)
}

func ConvertSecondsToMinutes(seconds int) int {
	return seconds / 60
}
