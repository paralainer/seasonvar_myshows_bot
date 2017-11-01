package app

import (
	"log"
	"gopkg.in/telegram-bot-api.v4"
	"strings"
	"strconv"
	"net/http"
	"fmt"
	"io"
	"os"
	"net/url"
)

type TgBot struct {
	Api      *tgbotapi.BotAPI
	Seasonvar *SeasonvarClient
}

func StartBot(token string, seasonvar *SeasonvarClient) {
	botApi, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		log.Panic(err)
	}

	botApi.Debug = true

	bot := &TgBot{
		Api:      botApi,
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
			parts := strings.Split(update.Message.Text, ":")
			if len(parts) == 3 {
				name := parts[0]
				seasonNum, _ := strconv.Atoi(parts[1])
				seriesNum, _ := strconv.Atoi(parts[2])
				seasons, e := bot.Seasonvar.SearchShow(name)
				if e != nil {
					log.Println(e)
				} else {
					found := false
					for _, season := range seasons {
						if season.Season == seasonNum {
							found = true
							link, err := bot.Seasonvar.GetDownloadLink(season.Id, seriesNum)
							if err != nil {
								log.Println(err)
							} else {
								message := tgbotapi.NewMessage(update.Message.Chat.ID, season.SerialName + " " + strconv.Itoa(season.Year))
								bot.Api.Send(message)

								message = tgbotapi.NewMessage(update.Message.Chat.ID, link.String())
								bot.Api.Send(message)
							}
							break
						}
					}

					if !found {
						message := tgbotapi.NewMessage(update.Message.Chat.ID, "Not found")
						bot.Api.Send(message)
					}

				}
			}
		}
	}


}

func saveFile(link *url.URL) int64 {
	response, err := http.Get(link.String())
	if err != nil {
		fmt.Println("Error while downloading", link, "-", err)
		return 0
	}
	defer response.Body.Close()


	output, err := os.Create("output/test.mp4")
	if err != nil {
		fmt.Println("Error while creating file", "-", err)
		return 0
	}
	defer output.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", link, "-", err)
		return 0
	}

	fmt.Println(n, "bytes downloaded.")
	return n
}



