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

type Season struct {
	ShowName     string
	SeasonNumber int
	Year         string
	SeasonId     int
}

type DownloadLink struct {
	Url         *url.URL
	Season      *Season
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
	var result []DownloadLink
	showName := dat["name_original"].(string)
	if showName == "" {
		showName = dat["name"].(string)
	}
	seasonSeries := dat["playlist"].([]interface{})
	year := dat["year"].(string)
	seasonNumber, err := strconv.Atoi(dat["season_number"].(string))
	if err != nil {
		seasonNumber = 0
	}
	season := Season {
		ShowName: showName,
		Year: year,
		SeasonId: seasonId,
		SeasonNumber: seasonNumber,
	}

	for _, s := range seasonSeries {
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
			Season: &season,
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

	var seasons []Season
	for _, s := range dat {
		season := s.(map[string]interface{})
		seasonNumber, err := strconv.Atoi(season["season"].([]interface{})[0].(string))
		if err != nil {
			return nil, err
		}

		year := season["year"].(string)
		id, _ := strconv.Atoi(season["id"].(string))
		seasons = append(seasons, Season{
			ShowName:     season["name"].(string),
			SeasonId:     id,
			Year:         year,
			SeasonNumber: seasonNumber,
		})
	}

	return seasons, nil
}