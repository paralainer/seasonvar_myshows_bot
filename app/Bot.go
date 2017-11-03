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

var MyShowsUnseenRegexp = regexp.MustCompile(`(.*) /show_\d+\n.*\ns(\d+)e(\d+).*`)
var MyShowsNewRegexp = regexp.MustCompile(`Новый эпизод сериала (.*)\n.*s(\d+)e(\d+).*`)
var MyShowsLinkRegexp = regexp.MustCompile(`https?://myshows\.me/view/episode/(\d+)/?`)
var SearchRegexp = regexp.MustCompile(`(.*):\s*(\d+)\s*:\s*(\d+)\s*`)

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

func (bot *TgBot) handleCallbackQuery(query *tgbotapi.CallbackQuery){
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
	go bot.sendSeasonEpisode(query.Message.Chat.ID, Season{
		Id: season,
	}, episode, false)
}

func (bot *TgBot) sendMatches(chatId int64, messageId int, matches []string){
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
	bot.sendEpisode(chatId, messageId, matches[1], season, episode)
}

func (bot *TgBot) handleMessage(chatId int64, messageId int, text string){
	matches := MyShowsUnseenRegexp.FindStringSubmatch(text)

	if  matches == nil {
		matches = MyShowsNewRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		matches = SearchRegexp.FindStringSubmatch(text)
	}

	if matches != nil {
		bot.sendMatches(chatId, messageId, matches)
	} else {
		linkSubmatch := MyShowsLinkRegexp.FindStringSubmatch(text)
		if linkSubmatch != nil {
			episodeId, _ := strconv.Atoi(linkSubmatch[1])
			episodeInfo := myshows.EpisodeById(episodeId)
			if episodeInfo != nil {
				bot.sendEpisode(chatId, messageId, episodeInfo.ShowName, episodeInfo.SeasonNumber, episodeInfo.EpisodeNumber)
			} else {
				bot.sendNotFound(chatId)
			}
		}
	}

}

func (bot *TgBot) sendEpisode(chatId int64, messageId int, query string, seasonNum int, episode int) {
	//message := tgbotapi.NewMessage(chatId, fmt.Sprintf("%s %d %d", name, seasonNum, episode))
	//bot.Api.Send(message)
	name := strings.TrimSpace(query)
	seasons, e := bot.Seasonvar.SearchShow(name)
	if e != nil {
		log.Println(e)
	} else {
		found := false

		matchedSeasons := getMatchedSeasons(name, seasons, seasonNum)
		if len(matchedSeasons) == 1 {
			found = bot.sendSeasonEpisode(chatId, matchedSeasons[0], episode, true)
		} else if len(matchedSeasons) > 1 {
			found = true
			bot.sendSeasonSelectionButtons(chatId, messageId, matchedSeasons, episode)
		}

		if !found {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) sendSeasonEpisode(chatId int64, season Season, episode int, sendText bool) bool{
	found := false
	links, err := bot.Seasonvar.GetDownloadLink(season.Id, episode)
	if err != nil {
		log.Println(err)
	} else {
		for _, link := range links {
			found = true
			if sendText {
				message := tgbotapi.NewMessage(
					chatId,
					fmt.Sprintf("%s %d %s",
						season.SerialName,
						season.Year,
						link.Translation))
				bot.Api.Send(message)
			} else {
				bot.Api.Send(tgbotapi.NewMessage(
					chatId,
					link.Translation))
			}

			linksMessage := tgbotapi.NewMessage(chatId, link.Url.String())
			bot.Api.Send(linksMessage)
		}
	}

	return found
}

func (bot *TgBot) sendSeasonSelectionButtons(chatId int64, messageId int, seasons []Season, episode int){
	buttons := [][]tgbotapi.InlineKeyboardButton{}
	for _, season := range seasons {
		button := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", season.SerialName, season.Year), fmt.Sprintf("SendById:%d:%d", season.Id, episode)))
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
		if strings.ToLower(season.SerialName) == strings.ToLower(query) {
			hasFullNameMatch = true
			break
		}
	}

	matchedSeasons := []Season{}
	for _, season := range seasons {
		if season.Season == seasonNum {
			if hasFullNameMatch {
				if strings.ToLower(season.SerialName) == strings.ToLower(query)  {
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
