package soap4me

import (
	"seasonvar_myshows_bot/app/types"
	"net/http"
	"encoding/json"
	"strconv"
	"net/url"
	"crypto/md5"
	"crypto/tls"
	"io/ioutil"
	"fmt"
)

const baseUrl = "https://api.soap4.me/v2"
const storageBaseUrl = "https://storage.soap4.me"


func NewApiClient(token string, session string) *ApiClient {
	return &ApiClient{
		authData: &authData{
			Token: token,
			Session: session,
		},
	}
}
type ApiClient struct {
	authData *authData
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func (client *ApiClient) fetchAuth() (*authData, error) {
	/*values := url.Values{}
	values.Add("login", client.login)
	values.Add("password", client.password)
	req, err := http.NewRequest(http.MethodPost, baseUrl+"/auth", nil)
	req.PostForm = values
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, &ApiClientError{err: "Invalid response code: " + strconv.Itoa(resp.StatusCode)}
	}

	var result authData
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, nil
	}
	return &result, nil*/

	return nil, nil
}

func (client *ApiClient) FindShows(query string) ([]types.TvShow, error) {
	req, err := http.NewRequest(http.MethodGet, baseUrl+"/search/?q="+query, nil)
	if err != nil {
		return nil, err
	}
	if err = client.auth(req); err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, &ApiClientError{err: "Invalid response code: " + strconv.Itoa(resp.StatusCode)}
	}
	var result searchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var shows []types.TvShow
	for _, show := range result.Series {
		shows = append(shows, *types.NewTvShow(show.Sid, show.Title, show.Title_ru))
	}

	return shows, nil
}

type ApiClientError struct {
	err string
}

func (e *ApiClientError) Error() string {
	return e.err
}

func (client *ApiClient) Episode(show types.TvShow, season int, episode int) (*types.Episode, error) {
	seasonS := strconv.Itoa(season)
	episodeS := strconv.Itoa(episode)
	req, err := http.NewRequest(http.MethodGet, baseUrl+"/episodes/"+show.Id + "/", nil)
	if err != nil {
		return nil, err
	}
	if err := client.auth(req); err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, &ApiClientError{err: "Invalid response code: " + strconv.Itoa(resp.StatusCode)}
	}
	var result episodesResult
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	for _, ep := range result.Episodes {
		if ep.Season == seasonS && ep.Episode == episodeS {
			return &types.Episode{
				TvShow:  show,
				Season:  season,
				Episode: episode,
				Links: func(files []file) []types.DownloadLink {
					var result []types.DownloadLink
					for _, file := range files {
						result = append(result, client.getLink(show, file))
					}
					return result
				}(ep.Files),
			}, nil
		}
	}

	return nil, nil
}

func qualityToString(quality string) string {
	switch quality {
	case "1":
		return types.QualityStandard
	case "2":
		return types.QualityHD
	case "3":
		return types.QualityFullHD
	case "4":
		return types.QualityUHD
	default:
		return types.QualityUnknown
	}
}
func translationToString(transaltion float64) string {
	switch t := int(transaltion); t {
	case 1:
		return types.AudioOriginal
	case 2:
		return types.AudioOriginalSubtitles
	case 3:
		return types.AudioLocalizedSubtitles
	case 4:
		return types.AudioLocalized
	default:
		return types.AudioUnknown
	}
}

func (client *ApiClient) getLink(show types.TvShow, file file) types.DownloadLink {
	return types.DownloadLink{
		Quality:     qualityToString(file.Quality),
		Translation: translationToString(file.Translate),
		Url:         *client.getDownloadUrl(show, file),
	}
}



func md5hash(text string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(text)))
}

func (client *ApiClient) auth(req *http.Request) error {
	if client.authData == nil {
		auth, err := client.fetchAuth()
		if err != nil {
			return err
		}
		client.authData = auth
	}
	req.Header.Set("X-Api-Token", client.authData.Token)
	req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: client.authData.Session})
	return nil
}

type searchResult struct {
	Series []tvShow `json:"series"`
}

type tvShow struct {
	Sid      string `json:"sid"`
	Title    string `json:"title"`
	Title_ru string `json:"title_ru"`
}

type episodesResult struct {
	Episodes []episode `json:"episodes"`
}

type episode struct {
	Season  string `json:"season"`
	Episode string `json:"episode"`
	Files   []file `json:"files"`
}

type file struct {
	Eid       string  `json:"eid"`
	Hash      string  `json:"hash"`
	Quality   string  `json:"quality"`
	Translate float64 `json:"translate"`
}

type authData struct{
	Token string `json:"token"`
	Session string `json:"session"`
}
