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

const apiEndpoint = "https://api.myshows.me/v2/rpc/"

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

	resp, err := http.Post(apiEndpoint, "application/json", strings.NewReader(request))

	if err != nil {
		fmt.Println(err)
		return nil
	}
	var dat map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&dat)

	if dat["result"] == nil {
		return nil
	}


	episodeJson := dat["result"].(map[string]interface{})
	episode := &EpisodeInfo{
		Id: id,
		EpisodeNumber: int(episodeJson["episodeNumber"].(float64)),
		SeasonNumber: int(episodeJson["seasonNumber"].(float64)),
		ShowId: int(episodeJson["showId"].(float64)),
	}

	fmt.Println(episode.ShowId)
	showName := fetchShowNameById(episode.ShowId)
	if showName == "" {
		return nil
	}

	episode.ShowName = showName
	fmt.Println(showName)

	return episode
}

func fetchShowNameById(showId int) string {
	request := fmt.Sprintf(`
		{
		  "jsonrpc": "2.0",
		  "method": "shows.GetById",
		  "params": {
			"showId": %d,
			"fetchEpisodes": false
		  },
		  "id": 1
		}
	`, showId)

	resp, err := http.Post(apiEndpoint, "application/json", strings.NewReader(request))

	if err != nil {
		fmt.Println(err)
		return ""
	}
	var dat map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&dat)

	if dat["result"] == nil {
		return ""
	}

	showJson := dat["result"].(map[string]interface{})

	return showJson["title"].(string)
}