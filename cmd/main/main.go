package main

import (
	"log"
	"os"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ttodoshi/weweather-bobot/internal/adapters/handler"
	"github.com/ttodoshi/weweather-bobot/internal/adapters/provider/http/openweathermap"
	"github.com/ttodoshi/weweather-bobot/internal/core/domain"
	"github.com/ttodoshi/weweather-bobot/internal/core/ports"
	"github.com/ttodoshi/weweather-bobot/pkg/env"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Функция инициализации, которая загружает переменные окружения
func init() {
	env.LoadEnvVariables()
}

func main() {
	// Создаем экземпляр бота с помощью токена из переменной окружения
	bot, err := tg.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Создаем экземпляр базы данных SQLite с именем "db.db" и мигрируем таблицы City и Notification
	db, err := gorm.Open(sqlite.Open("db.db"), &gorm.Config{})
	db.AutoMigrate(&domain.City{}, &domain.Notification{})

	// Создаем экземпляр провайдера погоды OpenWeatherMap с помощью токена API из переменной окружения
	weatherProvider := openweathermap.NewOpenWeatherMapWeatherProvider(os.Getenv("API_KEY"))
	// Создаем экземпляр обработчика команд
	commandHandler := handler.NewCommandHander(bot, db, weatherProvider)

	// Инициализируем сервис уведомлений
	initNotificationService(bot, db, weatherProvider)

	// Получаем канал обновлений (команд) от бота
	u := tg.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Цикл обработки обновлений (комманд)
	for update := range updates {
		commandHandler.HandleCommand(update)
	}
}

// Функция инициализации сервиса уведомлений, которая периодически проверяет текущее время и отправляет уведомления пользователю
func initNotificationService(bot *tg.BotAPI, db *gorm.DB, weatherProvider ports.WeatherProvider) {
	// Работает в отдельной горутине (виртуальном потоке)
	go func() {
		for {
			var notifications []domain.Notification
			db.Find(&notifications, "time = ?", time.Now().Format("15:04"))

			go func() {
				for _, notification := range notifications {
					msg := tg.NewMessage(notification.ChatID, "")

					var cities []string
					db.
						Select("city").
						Where("user_id = ?", notification.UserID).
						Table("cities").
						Find(&cities)
					if len(cities) > 0 {
						msg.Text = "Погода в ваших городах:\n"
						for _, w := range weatherProvider.FetchWeather(cities) {
							msg.Text += w
						}
					}

					// Отправляем оповещение
					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}
				}
			}()
			time.Sleep(60 * time.Second)
		}
	}()
}
