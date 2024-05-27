package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/ttodoshi/weweather-bobot/pkg/env"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Удалить город"),
	),
)

type City struct {
	UUID   string `gorm:"primaryKey"`
	UserID int64  `gorm:"not null;index"`
	City   string `gorm:"not null"`
}

func (e *City) BeforeCreate(_ *gorm.DB) (err error) {
	e.UUID = uuid.NewString()
	return
}

func main() {
	// Создаем экземпляр бота с помощью токена из переменной окружения
	bot, err := tg.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	db, err := gorm.Open(sqlite.Open("db.db"), &gorm.Config{})
	db.AutoMigrate(&City{})

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
				if city := strings.TrimSpace(update.Message.CommandArguments()); len(city) > 0 {
					msg.Text = "Добавлен город " + city
					db.Create(&City{
						UserID: update.Message.From.ID,
						City:   city,
					})
				} else {
					msg.Text = "Нельзя добавить город без названия"
				}
			case "delete":
				if city := strings.TrimSpace(update.Message.CommandArguments()); len(city) > 0 {
					msg.Text = "Удален город " + city
					db.Delete(&City{}, "user_id = ? AND city = ?", update.Message.From.ID, city)
				} else {
					msg.Text = "Нельзя удалить город без названия"
				}
			default:
				msg.Text = "Неизвестная команда"
			}
		} else {
			// Если сообщение не является командой
			switch update.Message.Text {
			case "Погода":
				msg.Text = "Погода в ваших городах:\n"
				var cities []string
				db.
					Select("city").
					Where("user_id = ?", update.Message.From.ID).
					Table("cities").
					Find(&cities)
				msg.Text += fetchWeather(cities)
			case "Добавить город":
				msg.Text = "Для добавления города напишите /add <Название города>"
			case "Удалить город":
				msg.Text = "Для удаления города напишите /delete <Название города>"
			}
		}

		// Отправляем сообщение
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func fetchWeather(cities []string) (response string) {
	// url := "https://wttr.in/{" + strings.Join(cities, ",") + "}?m2&lang=ru&format=4"
	// url := "https://wttr.in/{" + strings.Join(cities, ",") + "}?format=4"
	client := &http.Client{}

	for _, city := range cities {
		// url := `https://wttr.in/` + city + `?format=%l:+%c+🌡️+%t+%w\n`
		url := `https://wttr.in/` + city + `?format=4`
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		req.Header.Add("Accept-Language", "ru")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		response += string(body)
	}

	return
}
