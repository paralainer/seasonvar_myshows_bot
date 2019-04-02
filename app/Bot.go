package app

import (
	"fmt"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"regexp"
	"seasonvar_myshows_bot/app/myshows"
	"strconv"
	"strings"
)

type Handler func(bot *TgBot, chatId int64, matches []string)

type Strategy struct {
	Pattern *regexp.Regexp
	Name    string
	Handler Handler
}

func NewStrategy(name string, pattern *regexp.Regexp, handler Handler) *Strategy {
	return &Strategy{
		Name:    name,
		Pattern: pattern,
		Handler: handler,
	}
}

var IdRegexp = regexp.MustCompile(`id(\d+)\s+(\d+)\s*`)
var SeasonvarRegexp = regexp.MustCompile(`https?://seasonvar.ru/serial-(\d+).*\.html\s+(\d+)`)
var MobileSeasonvarRegexp = regexp.MustCompile(`https?://m.seasonvar.ru/#season/(\d+)\s+(\d+)`)
var MyShowsUnseenRegexp = regexp.MustCompile(`(.*) /show_\d+\n.*\ns(\d+)e(\d+).*`)
var MyShowsNewRegexp = regexp.MustCompile(`Новый эпизод сериала (.*)\n.*s(\d+)e(\d+).*`)
var MyShowsLinkRegexp = regexp.MustCompile(`https?://myshows\.me/view/episode/?(\d+)/?`)
var SearchRegexp = regexp.MustCompile(`(.*):\s*(\d+)\s*:\s*(\d+)\s*`)
var SearchSpacesRegexp = regexp.MustCompile(`(.*)\s+(\d+)\s+(\d+)\s*`)

var strategies = [...]*Strategy{
	NewStrategy("ById", IdRegexp, handleSeasonById),
	NewStrategy("SeasonvarLink", SeasonvarRegexp, handleSeasonById),
	NewStrategy("MobileSeasonvarLink", MobileSeasonvarRegexp, handleSeasonById),

	NewStrategy("MyShowsUnseenLink", MyShowsUnseenRegexp, handleSearchForEpisode),
	NewStrategy("MyShowsNewEpisode", MyShowsNewRegexp, handleSearchForEpisode),

	NewStrategy("MyShowsLink", MyShowsLinkRegexp, handleMyShowsLink),

	NewStrategy("SearchColons", SearchRegexp, handleSearchForEpisode),
	NewStrategy("SearchSpaces", SearchSpacesRegexp, handleSearchForEpisode),
}

func handleSearchForEpisode(bot *TgBot, chatId int64, matches []string) {
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

func handleMyShowsLink(bot *TgBot, chatId int64, matches []string) {
	episodeId, _ := strconv.Atoi(matches[1])
	episodeInfo := myshows.EpisodeById(episodeId)
	if episodeInfo != nil {
		bot.sendEpisode(chatId, episodeInfo.ShowName, episodeInfo.SeasonNumber, episodeInfo.EpisodeNumber)
	} else {
		bot.sendNotFound(chatId)
	}
}

func handleSeasonById(bot *TgBot, chatId int64, matches []string) {
	seasonId, _ := strconv.Atoi(matches[1])
	episode, _ := strconv.Atoi(matches[2])
	go bot.sendSeasonEpisode(chatId, seasonId, episode)
}

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
			go bot.handleMessage(chatId, text)
		} else if update.CallbackQuery != nil {
			go bot.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (bot *TgBot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	queryParts := strings.Split(query.Data, ":")
	seasonId, e := strconv.Atoi(queryParts[1])
	if e != nil {
		log.Println(e)
		return
	}

	episode, e := strconv.Atoi(strings.TrimSpace(queryParts[3]))
	if e != nil {
		log.Println(e)
		return
	}
	go bot.sendSeasonEpisode(query.Message.Chat.ID, seasonId, episode)
}

func (bot *TgBot) handleMessage(chatId int64, text string) {
	for _, strategy := range strategies {
		matches := strategy.Pattern.FindStringSubmatch(text)
		if matches != nil {
			log.Println("Using strategy: " + strategy.Name)
			strategy.Handler(bot, chatId, matches)
		}
	}
}

func (bot *TgBot) sendEpisode(chatId int64, query string, seasonNum int, episode int) {
	name := strings.TrimSpace(query)
	seasons, e := bot.Seasonvar.SearchShow(name)
	if e != nil {
		log.Println(e)
	} else {

		matchedSeasons := getMatchedSeasons(name, seasons, seasonNum)
		if len(matchedSeasons) == 1 {
			go bot.sendSeasonEpisode(chatId, matchedSeasons[0].SeasonId, episode)
		} else if len(matchedSeasons) > 1 {
			go bot.sendSeasonSelectionButtons(chatId, matchedSeasons, episode)
		}

		if len(matchedSeasons) == 0 {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) sendSeasonEpisode(chatId int64, seasonId int, episode int) {
	found := false
	links, err := bot.Seasonvar.GetDownloadLink(seasonId, episode)
	if err != nil {
		log.Println(err)
	} else {
		var translations []string
		for _, link := range links {
			found = true
			text := fmt.Sprintf("%s %s ",
				link.Season.PrintableName(),
				link.Season.Year)
			translations = append(translations, fmt.Sprintf("%ss%02de%02d [%s](%s)", text, link.Season.SeasonNumber, episode, link.Translation, link.Url.String()))
		}

		if found {
			_, _ = bot.Api.Send(tgbotapi.MessageConfig{
				BaseChat: tgbotapi.BaseChat{
					ChatID:           chatId,
					ReplyToMessageID: 0,
					ReplyMarkup:      getNextAndPreviousButtons(seasonId, links[0].Season.SeasonNumber, episode),
				},
				Text:                  strings.Join(translations, "\n\n"),
				DisableWebPagePreview: false,
				ParseMode:             tgbotapi.ModeMarkdown,
			})
		}
	}

	if !found {
		bot.sendNotFound(chatId)
	}
}

func (bot *TgBot) sendSeasonSelectionButtons(chatId int64, seasons []Season, episode int) {
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, season := range seasons {
		button := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", season.PrintableName(), season.Year), fmt.Sprintf("SendById:%d:%d:%d", season.SeasonId, season.SeasonNumber, episode)))
		buttons = append(buttons, button)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	message := tgbotapi.NewMessage(chatId, "Select tv show")
	message.ReplyMarkup = markup
	_, _ = bot.Api.Send(message)
}

func getNextAndPreviousButtons(seasonId int, seasonNumber int, currentEpisode int) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	nextButton := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Next Episode"), fmt.Sprintf("SendById:%d:%d:%d", seasonId, seasonNumber, currentEpisode+1)))
	previousButton := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Previous Episode"), fmt.Sprintf("SendById:%d:%d:%d", seasonId, seasonNumber, currentEpisode-1)))
	buttons = append(buttons, nextButton)
	buttons = append(buttons, previousButton)

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func getMatchedSeasons(query string, seasons []Season, seasonNum int) []Season {
	hasFullNameMatch := false
	normalizedQuery := strings.ToLower(query)

	matchAlternativeNames := func(names []string) bool {
		for _, name := range names {
			if strings.ToLower(name) == normalizedQuery {
				return true
			}
		}
		return false
	}

	isNameMatches := func(season *Season) bool {
		return strings.ToLower(season.ShowName) == normalizedQuery ||
			strings.ToLower(season.ShowOriginalName) == normalizedQuery ||
			matchAlternativeNames(season.ShowAlternativeNames)
	}

	for _, season := range seasons {
		if isNameMatches(&season) {
			hasFullNameMatch = true
			break
		}
	}

	var matchedSeasons []Season
	for _, season := range seasons {
		if season.SeasonNumber == seasonNum {
			if hasFullNameMatch {
				if isNameMatches(&season) {
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
	_, _ = bot.Api.Send(message)
}
