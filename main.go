package main

import (
	"encoding/json"
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
	weatherService := NewOpenWeatherMapWeatherService(os.Getenv("API_KEY"))
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
				var cities []string
				db.
					Select("city").
					Where("user_id = ?", update.Message.From.ID).
					Table("cities").
					Find(&cities)
				if len(cities) > 0 {
					msg.Text = "Погода в ваших городах:\n"
					for _, w := range weatherService.fetchWeather(cities) {
						msg.Text += w
					}
				} else {
					msg.Text = "У вас ещё нет отслеживаемых городов"
				}
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

type WeatherService interface {
	fetchWeather(cities []string) []string
}

type wttrInWeatherService struct{}

func NewWttrInWeatherService() WeatherService {
	return &wttrInWeatherService{}
}

func (s *wttrInWeatherService) fetchWeather(cities []string) (response []string) {
	client := &http.Client{}

	for _, city := range cities {
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
		response = append(response, string(body))
	}

	return
}

type openWeatherMapWeatherService struct {
	apiKey string
}

func NewOpenWeatherMapWeatherService(apiKey string) WeatherService {
	return &openWeatherMapWeatherService{
		apiKey: apiKey,
	}
}

func (s *openWeatherMapWeatherService) fetchWeather(cities []string) (response []string) {
	client := &http.Client{}

	for _, city := range cities {
		locationUrl := fmt.Sprintf(
			"http://api.openweathermap.org/geo/1.0/direct?q=%s&appid=%s&limit=1",
			city,
			s.apiKey,
		)
		req, err := http.NewRequest("GET", locationUrl, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

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
		var cityLocations []map[string]any
		err = json.Unmarshal(body, &cityLocations)
		if err != nil {
			fmt.Println(err)
			return
		}
		cityLocation := cityLocations[0]

		url := fmt.Sprintf(
			"https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ru",
			cityLocation["lat"].(float64),
			cityLocation["lon"].(float64),
			s.apiKey,
		)
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		resp, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		var res map[string]any
		err = json.Unmarshal(body, &res)
		if err != nil {
			fmt.Println(err)
			return
		}

		response = append(
			response,
			fmt.Sprintf(
				"%s: %s 🌡️%.1f⁰C 🌬️ %.1f km/h\n",
				city,
				res["weather"].([]any)[0].(map[string]any)["description"].(string),
				res["main"].(map[string]any)["temp"].(float64),
				res["wind"].(map[string]any)["speed"].(float64),
			),
		)
	}

	return
}
