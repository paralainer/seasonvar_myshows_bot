package app

import (
	"net/url"
	"net/http"
	"strconv"
	"encoding/json"
	"log"
	"io/ioutil"
)

const apiUrl = "http://api.seasonvar.ru"

type Season struct{
	SerialName string
	Season int
	Year int
	Id int
}

type SeasonvarClient struct {
	ApiToken string
}

func (sc *SeasonvarClient) postParams() *url.Values {
	values := &url.Values{}
	values.Add("key", sc.ApiToken)
	return values
}


func (sc *SeasonvarClient) GetDownloadLink(seasonId int, seriesNumber int) (*url.URL, error) {
	params := sc.postParams()
	params.Add("command", "getSeason")
	params.Add("season_id", strconv.Itoa(seasonId))
	resp, err := http.PostForm(apiUrl, *params)
	if err != nil {
		return nil, err
	}

	var dat map[string]interface{}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &dat); err != nil {
		log.Println("Error parsing json")
		return nil, err
	}

	series := dat["playlist"].([]interface{})
	linkString := series[seriesNumber].(map[string]interface{})["link"].(string)
	link, err := url.Parse(linkString)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (sc *SeasonvarClient) SearchShow(query string) ([]Season, error) {
	params := sc.postParams()
	params.Add("command", "search")
	params.Add("query", query)
	resp, err := http.PostForm(apiUrl, *params)
	if err != nil {
		return nil, err
	}

	var dat []interface{}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &dat); err != nil {
		log.Println("Error parsing json")
		return nil, err
	}

	seasons := []Season{}
	for _, s := range dat {
		season := s.(map[string]interface{})
		seasonNumber, err := strconv.Atoi(season["season"].([]interface{})[0].(string))
		if err != nil {
			return nil, err
		}

		year, _ := strconv.Atoi(season["year"].(string))
		id, _ := strconv.Atoi(season["id"].(string))
		seasons = append(seasons, Season{
			SerialName: season["name"].(string),
			Season: seasonNumber,
			Year: year,
			Id: id,
		})
	}

	return seasons, nil
}