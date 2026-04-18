package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("🤖 Bot started:", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		// 👉 Handle /start
		if update.Message.Text == "/start" {

			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				"🎯 Welcome to Bingo!\nClick below to play 👇",
			)

			bot.Send(msg)
		}
	}
}