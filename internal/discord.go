package internal

import (
	"fmt"
	"time"

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

	if anime.EpisodeDuration == 0 {
		dur, err := getVideoDuration("/tmp/iinasocket")
		if err != nil {
			return err
		}
		anime.EpisodeDuration = int(dur)
	}

	var state string
	var startTime *time.Time
	var endTime *time.Time

	if isPaused {
		state = fmt.Sprintf("\nEpisode %d (Paused)", anime.CurrentEpisode)
		startTime = nil
		endTime = nil
	} else {
		state = fmt.Sprintf("\nEpisode %d", anime.CurrentEpisode)
		now := time.Now()
		calculatedStartTime := now.Add(-time.Duration(timePos) * time.Second)
		calculatedEndTime := calculatedStartTime.Add(time.Duration(anime.EpisodeDuration) * time.Second)
		startTime = &calculatedStartTime
		endTime = &calculatedEndTime
	}

	err = client.SetActivity(client.Activity{
		Type:       3,
		Details:    anime.Title,
		LargeImage: anime.CoverImage,
		LargeText:  anime.Title,
		State:      state,
		Timestamps: &client.Timestamps{
			Start: startTime,
			End:   endTime,
		},
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
