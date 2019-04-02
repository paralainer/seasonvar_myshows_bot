package main

import (
	"log"
	"os"
	"seasonvar_myshows_bot/app"
)

func main() {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Panic("No TELEGRAM_TOKEN specified")
	}

	seasonvarToken := os.Getenv("SEASONVAR_TOKEN")
	if telegramToken == "" {
		log.Panic("No SEASONVAR_TOKEN specified")
	}

	app.StartBot(telegramToken, &app.SeasonvarClient{ApiToken: seasonvarToken})
}
