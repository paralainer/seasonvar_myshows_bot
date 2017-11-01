package app

import (
	"net/url"
	"net/http"
	"strconv"
	"encoding/json"
	"log"
	"io/ioutil"
	"strings"
)

const apiUrl = "http://api.seasonvar.ru"

type Season struct{
	SerialName string
	Season int
	Year int
	Id int
}

type DownloadLink struct {
	Url *url.URL
	Translation string
}

type SeasonvarClient struct {
	ApiToken string
}

func (sc *SeasonvarClient) postParams() *url.Values {
	values := &url.Values{}
	values.Add("key", sc.ApiToken)
	return values
}


func (sc *SeasonvarClient) GetDownloadLink(seasonId int, seriesNumber int) ([]DownloadLink, error) {
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
	result := []DownloadLink{}
	season := dat["playlist"].([]interface{})
	log.Println(season)
	for _, s := range season {
		series := s.(map[string]interface{})
		num, err := strconv.Atoi(strings.Split(series["name"].(string), " ")[0])
		if err != nil {
			continue
		}

		if num != seriesNumber {
			continue
		}

		linkString := series["link"].(string)
		link, err := url.Parse(linkString)
		if err != nil {
			return nil, err
		}

		t := series["perevod"]
		translation := "Original"
		if t != nil {
			translation = t.(string)
		}

		result = append(result, DownloadLink{
			Url: link,
			Translation: translation,
		})

	}


	return result, nil
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