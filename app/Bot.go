package app

import (
	"log"
	"gopkg.in/telegram-bot-api.v4"
	"strings"
	"regexp"
	"seasonvar_myshows_bot/app/soap4me"
	"strconv"
	"seasonvar_myshows_bot/app/myshows"
	"fmt"
	"seasonvar_myshows_bot/app/types"
)

var MyShowsUnseenRegexp = regexp.MustCompile(`(.*) /show_\d+\n.*\ns(\d+)e(\d+).*`)
var MyShowsNewRegexp = regexp.MustCompile(`Новый эпизод сериала (.*)\n.*s(\d+)e(\d+).*`)
var MyShowsLinkRegexp = regexp.MustCompile(`https?://myshows\.me/view/episode/(\d+)/?`)
var SearchRegexp = regexp.MustCompile(`(.*):\s*(\d+)\s*:\s*(\d+)\s*`)
var SearchSpacesRegexp = regexp.MustCompile(`(.*)\s+(\d+)\s+(\d+)\s*`)

type TgBot struct {
	Api     *tgbotapi.BotAPI
	Soap4Me *soap4me.ApiClient
}

func StartBot(token string, soap4Me *soap4me.ApiClient) {
	botApi, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		log.Panic(err)
	}

	botApi.Debug = false

	bot := &TgBot{
		Api:     botApi,
		Soap4Me: soap4Me,
	}

	bot.startBot()
}

func (bot *TgBot) startBot() {
	log.Printf("Authorized on account %s", bot.Api.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.Api.GetUpdatesChan(u)

	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message != nil {
			text := update.Message.Text
			log.Println("Query:" + text)
			chatId := update.Message.Chat.ID
			go bot.handleMessage(chatId, update.Message.MessageID, text)
		} else if update.CallbackQuery != nil {
			go bot.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (bot *TgBot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	queryParts := strings.Split(query.Data, ":")
	if len(queryParts) != 4 {
		return
	}
	tvShowId := queryParts[1]

	season, e := strconv.Atoi(strings.TrimSpace(queryParts[2]))
	if e != nil {
		log.Println(e)
		return
	}

	episode, e := strconv.Atoi(strings.TrimSpace(queryParts[3]))
	if e != nil {
		log.Println(e)
		return
	}

	bot.sendEpisode(query.Message.Chat.ID, *types.NewTvShow(tvShowId, "", ""), season, episode, false)
}

func (bot *TgBot) sendMatches(chatId int64, messageId int, matches []string) {
	season, e := strconv.Atoi(strings.TrimSpace(matches[2]))
	if e != nil {
		log.Println(e)
		return
	}

	episode, e := strconv.Atoi(strings.TrimSpace(matches[3]))
	if e != nil {
		log.Println(e)
		return
	}
	bot.findAndSendEpisode(chatId, messageId, matches[1], season, episode)
}

func (bot *TgBot) handleMessage(chatId int64, messageId int, text string) {
	matches := MyShowsUnseenRegexp.FindStringSubmatch(text)

	if matches == nil {
		matches = MyShowsNewRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		matches = SearchRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		matches = SearchSpacesRegexp.FindStringSubmatch(text)
	}

	if matches != nil {
		bot.sendMatches(chatId, messageId, matches)
	} else {
		linkSubmatch := MyShowsLinkRegexp.FindStringSubmatch(text)
		if linkSubmatch != nil {
			episodeId, _ := strconv.Atoi(linkSubmatch[1])
			episodeInfo := myshows.EpisodeById(episodeId)
			if episodeInfo != nil {
				bot.findAndSendEpisode(chatId, messageId, episodeInfo.ShowName, episodeInfo.SeasonNumber, episodeInfo.EpisodeNumber)
			} else {
				bot.sendNotFound(chatId)
			}
		}
	}

}

func (bot *TgBot) findAndSendEpisode(chatId int64, messageId int, query string, seasonNum int, episode int) {
	//message := tgbotapi.NewMessage(chatId, fmt.Sprintf("%s %d %d", name, seasonNum, episode))
	//bot.Api.Send(message)
	name := strings.TrimSpace(query)
	seasons, e := bot.Soap4Me.FindShows(name)
	if e != nil {
		log.Println(e)
	} else {
		found := false

		matchedShows := getMatchedShows(name, seasons)
		if len(matchedShows) == 1 {
			found = bot.sendEpisode(chatId, matchedShows[0], seasonNum, episode, true)
		} else if len(matchedShows) > 1 {
			found = true
			bot.sendSeasonSelectionButtons(chatId, messageId, matchedShows, seasonNum, episode)
		}

		if !found {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) sendEpisode(chatId int64, tvShow types.TvShow, seasonNum int, episodeNum int, sendText bool) bool {
	episode, err := bot.Soap4Me.Episode(tvShow, seasonNum, episodeNum)
	if err != nil {
		log.Println(err)
		return false
	}
	if episode == nil || len(episode.Links) == 0 {
		return false
	}
	var lines []string
	if sendText {
		lines = append(lines, fmt.Sprintf("%s / %s s%02de%02d", tvShow.Name, tvShow.LocalizedName, episode.Season, episode.Episode), "")
	}

	for _, translation := range []string{types.AudioOriginal, types.AudioOriginalSubtitles, types.AudioLocalizedSubtitles, types.AudioLocalized, types.AudioUnknown} {
		var links []types.DownloadLink
		for _, link := range episode.Links {
			if link.Translation == translation {
				links = append(links, link)
			}
		}
		if len(links) > 0 {
			var buttons []tgbotapi.InlineKeyboardButton
			for _, link := range links {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%s / %s", tvShow.Name, tvShow.LocalizedName),
					fmt.Sprintf("PlaySoapEpisode:%s:%d:%d", ),
				))
			}
			row := tgbotapi.NewInlineKeyboardRow()
			markup := tgbotapi.NewInlineKeyboardMarkup(row)
			message := tgbotapi.NewMessage(chatId, translation)
			message.ReplyMarkup = markup
			bot.Api.Send(message)
		}
	}
	for _, link := range episode.Links {
		lines = append(lines, fmt.Sprintf("%s - %s:", link.Translation, link.Quality), link.Url.String(), "")
	}

	message := tgbotapi.NewMessage(
		chatId,
		strings.Join(lines, "\n"),
	)

	bot.Api.Send(message)
	return true
}

func (bot *TgBot) sendSeasonSelectionButtons(chatId int64, messageId int, tvShows []types.TvShow, season int, episode int) {
	buttons := [][]tgbotapi.InlineKeyboardButton{}
	for _, tvShow := range tvShows {
		button := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s / %s", tvShow.Name, tvShow.LocalizedName),
				fmt.Sprintf("SendById:%s:%d:%d", tvShow.Id, season, episode),
			),
		)
		buttons = append(buttons, button)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	message := tgbotapi.NewMessage(chatId, "Select tv show")
	message.ReplyMarkup = markup
	bot.Api.Send(message)
}

func getMatchedShows(query string, tvShows []types.TvShow) []types.TvShow {
	query = strings.ToLower(query)
	for _, tvShow := range tvShows {
		if strings.ToLower(tvShow.Name) == query || strings.ToLower(tvShow.LocalizedName) == query {
			return []types.TvShow{tvShow}
		}
	}

	return tvShows
}

func (bot *TgBot) sendNotFound(chatId int64) {
	message := tgbotapi.NewMessage(chatId, "Not found")
	bot.Api.Send(message)
}
