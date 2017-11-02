package app

import (
	"log"
	"gopkg.in/telegram-bot-api.v4"
	"strings"
	"strconv"
	"regexp"
	"fmt"
)

var MyShowsUnseenRegexp = regexp.MustCompile(`(.*) /show_\d+\n.*\ns(\d+)e(\d+).*`)
var MyShowsNewRegexp = regexp.MustCompile(`Новый эпизод сериала (.*)\n.*s(\d+)e(\d+).*`)
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
			matches := MyShowsUnseenRegexp.FindStringSubmatch(text)

			if  matches == nil {
				matches = MyShowsNewRegexp.FindStringSubmatch(text)
			}

			if matches == nil {
				matches = SearchRegexp.FindStringSubmatch(text)
			}

			if matches != nil {
				go bot.sendSeries(chatId, matches[1], matches[2], matches[3])
			}
		}
	}
}

func (bot *TgBot) sendSeries(chatId int64, name string, seasonNumS string, seriesNumS string) {
	seasonNum, e := strconv.Atoi(strings.TrimSpace(seasonNumS))
	if e != nil {
		log.Println(e)
		return
	}

	seriesNum, e := strconv.Atoi(strings.TrimSpace(seriesNumS))
	if e != nil {
		log.Println(e)
		return
	}

	seasons, e := bot.Seasonvar.SearchShow(name)
	if e != nil {
		log.Println(e)
	} else {
		found := false
		for _, season := range seasons {
			if season.Season == seasonNum {
				links, err := bot.Seasonvar.GetDownloadLink(season.Id, seriesNum)
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
			message := tgbotapi.NewMessage(chatId, "Not found")
			bot.Api.Send(message)
		}
	}
}
