package main

import (
	"seasonvar_myshows_bot/app"
	"os"
	"log"
	"seasonvar_myshows_bot/app/soap4me"
)

func main() {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Panic("No TELEGRAM_TOKEN specified")
	}

	token := os.Getenv("SOAP_TOKEN")
	if token == "" {
		log.Panic("No SOAP_TOKEN specified")
	}

	session := os.Getenv("SOAP_SESSION")
	if session == "" {
		log.Panic("No SOAP_SESSION specified")
	}

	app.StartBot(telegramToken, soap4me.NewApiClient(token, session));
}

