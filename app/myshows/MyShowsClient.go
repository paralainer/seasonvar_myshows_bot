package myshows

import (
	"fmt"
	"net/http"
	"strings"
	"encoding/json"
)

type EpisodeInfo struct {
	Id int
	ShowId int
	ShowName string
	SeasonNumber int
	EpisodeNumber int
}

const myShowsApiEndpoint = "https://api.myshows.me/v2/rpc/"

func EpisodeById(id int) *EpisodeInfo {
	request := fmt.Sprintf(`
		{
		  "jsonrpc": "2.0",
		  "method": "shows.Episode",
		  "params": {
			"id": %d
		  },
		  "id": 1
		}
	`, id)

	resp, err := http.Post(myShowsApiEndpoint, "application/json", strings.NewReader(request))

	if err != nil {
		fmt.Println(err)
		return nil
	}
	var dat map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&dat)

	if dat["result"] == nil {
		return nil
	}

	episode := &EpisodeInfo{}
	episodeJson := dat["result"].(map[string]interface{})
	episode.Id = id
	episode.EpisodeNumber = episodeJson["episodeNumber"].(int)
	episode.SeasonNumber = episodeJson["seasonNumber"].(int)
	episode.ShowId = episodeJson["showId"].(int)

	showName := fetchShowNameById(episode.ShowId)
	if showName == nil {
		return nil
	}

	episode.ShowName = *showName

	return episode
}

func fetchShowNameById(showId int) *string {
	request := fmt.Sprintf(`
		{
		  "jsonrpc": "2.0",
		  "method": "shows.Episode",
		  "params": {
			"showId": %d,
			"fetchEpisodes": false
		  },
		  "id": 1
		}
	`, showId)

	resp, err := http.Post(myShowsApiEndpoint, "application/json", strings.NewReader(request))

	if err != nil {
		fmt.Println(err)
		return nil
	}
	var dat map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&dat)

	if dat["result"] == nil {
		return nil
	}

	showJson := dat["result"].(map[string]interface{})

	return &showJson["title"].(string)
}