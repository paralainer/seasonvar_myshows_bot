package app

import (
	"log"
	"gopkg.in/telegram-bot-api.v4"
	"strings"
	"strconv"
	"regexp"
	"fmt"
	"seasonvar_myshows_bot/app/myshows"
)

var IdRegexp = regexp.MustCompile(`id(\d+)\s+(\d+)\s*`)
var SeasonvarRegexp = regexp.MustCompile(`https?://seasonvar.ru/serial-(\d+).*\.html\s+(\d+)`)
var MobileSeasonvarRegexp = regexp.MustCompile(`https?://m.seasonvar.ru/#season/(\d+).*\.html\s+(\d+)`)
var MyShowsUnseenRegexp = regexp.MustCompile(`(.*) /show_\d+\n.*\ns(\d+)e(\d+).*`)
var MyShowsNewRegexp = regexp.MustCompile(`Новый эпизод сериала (.*)\n.*s(\d+)e(\d+).*`)
var MyShowsLinkRegexp = regexp.MustCompile(`https?://myshows\.me/view/episode/(\d+)/?`)
var SearchRegexp = regexp.MustCompile(`(.*):\s*(\d+)\s*:\s*(\d+)\s*`)
var SearchSpacesRegexp = regexp.MustCompile(`(.*)\s+(\d+)\s+(\d+)\s*`)

type TgBot struct {
	Api       *tgbotapi.BotAPI
	Seasonvar *SeasonvarClient
}

func StartBot(token string, seasonvar *SeasonvarClient) {
	botApi, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		log.Panic(err)
	}

	botApi.Debug = false

	bot := &TgBot{
		Api:       botApi,
		Seasonvar: seasonvar,
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
	season, e := strconv.Atoi(queryParts[1])
	if e != nil {
		log.Println(e)
		return
	}

	episode, e := strconv.Atoi(strings.TrimSpace(queryParts[2]))
	if e != nil {
		log.Println(e)
		return
	}
	go bot.sendSeasonEpisode(query.Message.Chat.ID, season, episode, false)
}

func (bot *TgBot) sendMatches(chatId int64, matches []string) {
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

	bot.sendEpisode(chatId, matches[1], season, episode)
}

func (bot *TgBot) handleMessage(chatId int64, messageId int, text string) {

	if bot.handleSeasonById(text, chatId) {
		return
	}

	if bot.handleSearchForSeason(text, chatId) {
		return
	}

	linkSubmatch := MyShowsLinkRegexp.FindStringSubmatch(text)
	if linkSubmatch != nil {
		episodeId, _ := strconv.Atoi(linkSubmatch[1])
		episodeInfo := myshows.EpisodeById(episodeId)
		if episodeInfo != nil {
			bot.sendEpisode(chatId, episodeInfo.ShowName, episodeInfo.SeasonNumber, episodeInfo.EpisodeNumber)
		} else {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) handleSearchForSeason(text string, chatId int64) bool {
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

	if matches == nil {
		return false
	}

	bot.sendMatches(chatId, matches)

	return true
}

func (bot *TgBot) handleSeasonById(text string, chatId int64) bool {
	matches := IdRegexp.FindStringSubmatch(text)

	if matches == nil {
		matches = SeasonvarRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		matches = MobileSeasonvarRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		return false
	}

	seasonId, _ := strconv.Atoi(matches[1])
	episode, _ := strconv.Atoi(matches[2])
	bot.sendSeasonEpisode(chatId, seasonId, episode, true)

	return true
}

func (bot *TgBot) sendEpisode(chatId int64, query string, seasonNum int, episode int) {
	name := strings.TrimSpace(query)
	seasons, e := bot.Seasonvar.SearchShow(name)
	if e != nil {
		log.Println(e)
	} else {
		found := false

		matchedSeasons := getMatchedSeasons(name, seasons, seasonNum)
		if len(matchedSeasons) == 1 {
			found = bot.sendSeasonEpisode(chatId, matchedSeasons[0].SeasonId, episode, true)
		} else if len(matchedSeasons) > 1 {
			found = true
			bot.sendSeasonSelectionButtons(chatId, matchedSeasons, episode)
		}

		if !found {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) sendSeasonEpisode(chatId int64, seasonId int, episode int, sendText bool) bool {
	found := false
	links, err := bot.Seasonvar.GetDownloadLink(seasonId, episode)
	if err != nil {
		log.Println(err)
	} else {
		var translations []string
		for _, link := range links {
			found = true
			text := ""
			if sendText {
				text =
					fmt.Sprintf("%s %s ",
						link.Season.ShowName,
						link.Season.Year)
			}
			translations = append(translations, fmt.Sprintf("%s[%s](%s)", text, link.Translation, link.Url.String()))
		}

		bot.Api.Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           chatId,
				ReplyToMessageID: 0,
			},
			Text:                  strings.Join(translations, "\n\n"),
			DisableWebPagePreview: false,
			ParseMode:             tgbotapi.ModeMarkdown,
		})
	}

	return found
}

func (bot *TgBot) sendSeasonSelectionButtons(chatId int64, seasons []Season, episode int) {
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, season := range seasons {
		button := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", season.ShowName, season.Year), fmt.Sprintf("SendById:%d:%d", season.SeasonId, episode)))
		buttons = append(buttons, button)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	message := tgbotapi.NewMessage(chatId, "Select tv show")
	message.ReplyMarkup = markup
	bot.Api.Send(message)
}

func getMatchedSeasons(query string, seasons []Season, seasonNum int) []Season {
	hasFullNameMatch := false
	for _, season := range seasons {
		if strings.ToLower(season.ShowName) == strings.ToLower(query) {
			hasFullNameMatch = true
			break
		}
	}

	var matchedSeasons []Season
	for _, season := range seasons {
		if season.SeasonId == seasonNum {
			if hasFullNameMatch {
				if strings.ToLower(season.ShowName) == strings.ToLower(query) {
					matchedSeasons = append(matchedSeasons, season)
				}
			} else {
				matchedSeasons = append(matchedSeasons, season)
			}
		}
	}

	return matchedSeasons
}

func (bot *TgBot) sendNotFound(chatId int64) {
	message := tgbotapi.NewMessage(chatId, "Not found")
	bot.Api.Send(message)
}
