package main

import (
	"log"
	"os"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ttodoshi/weweather-bobot/pkg/env"
)

// Функция инициализации, которая загружает переменные окружения
func init() {
	env.LoadEnvVariables()
}

// Определение основной клавиатуры для бота
var mainKeyboard = tg.NewReplyKeyboard(
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Погода"),
	),
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Добавить город"),
	),
)

func main() {
	// Создаем экземпляр бота с помощью токена из переменной окружения
	bot, err := tg.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Получаем канал обновлений (команд) от бота
	u := tg.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Цикл обработки обновлений (комманд)
	for update := range updates {
		// Если обновление не содержит содержимого, то игнорируем
		if update.Message == nil {
			continue
		}

		// Создаем новое сообщение для ответа
		msg := tg.NewMessage(update.Message.Chat.ID, "")

		// Если сообщение является командой
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg.ReplyMarkup = mainKeyboard
				msg.Text = "Привет, Я телеграм-бот для погоды. Нажми на кнопки интересующую тебя команду ниже:"
			case "add":
				msg.Text = "Добавлен город " + update.Message.CommandArguments()
			default:
				msg.Text = "Неизвестная команда"
			}
		} else {
			// Если сообщение не является командой
			switch update.Message.Text {
			case "Погода":
				msg.Text = "Погода в ваших городах:"
			default:
				msg.Text = "Для добавления города напишите /add <Название города>"
			}
		}

		// Отправляем сообщение
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
