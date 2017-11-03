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
			go bot.handleMessage(chatId, text)
		}
	}
}

func (bot *TgBot) handleMessage(chatId int64, text string){
	sendMatches := func(chatId int64, matches []string) {
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
		bot.sendSeries(chatId, matches[1], season, episode)
	}

	matches := MyShowsUnseenRegexp.FindStringSubmatch(text)

	if  matches == nil {
		matches = MyShowsNewRegexp.FindStringSubmatch(text)
	}

	if matches == nil {
		matches = SearchRegexp.FindStringSubmatch(text)
	}

	if matches != nil {
		sendMatches(chatId, matches)
	} else {
		linkSubmatch := MyShowsLinkRegexp.FindStringSubmatch(text)
		if linkSubmatch != nil {
			episodeId, _ := strconv.Atoi(linkSubmatch[1])
			episodeInfo := myshows.EpisodeById(episodeId)
			if episodeInfo != nil {
				bot.sendSeries(chatId, episodeInfo.ShowName, episodeInfo.SeasonNumber, episodeInfo.EpisodeNumber)
			} else {
				bot.sendNotFound(chatId)
			}
		}
	}

}

func (bot *TgBot) sendSeries(chatId int64, name string, seasonNum int, episode int) {
	//message := tgbotapi.NewMessage(chatId, fmt.Sprintf("%s %d %d", name, seasonNum, episode))
	//bot.Api.Send(message)
	seasons, e := bot.Seasonvar.SearchShow(name)
	if e != nil {
		log.Println(e)
	} else {
		found := false
		for _, season := range seasons {
			if season.Season == seasonNum {
				links, err := bot.Seasonvar.GetDownloadLink(season.Id, episode)
				if err != nil {
					log.Println(err)
				} else {
					for _, link := range links {
						found = true

						message := tgbotapi.NewMessage(
							chatId,
							fmt.Sprintf("%s %d %s",
								season.SerialName,
								season.Year,
								link.Translation))
						bot.Api.Send(message)

						message = tgbotapi.NewMessage(chatId, link.Url.String())
						bot.Api.Send(message)
					}
				}
				break
			}
		}

		if !found {
			bot.sendNotFound(chatId)
		}
	}
}

func (bot *TgBot) sendNotFound(chatId int64) {
	message := tgbotapi.NewMessage(chatId, "Not found")
	bot.Api.Send(message)
}
