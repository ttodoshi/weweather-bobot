package handler

import (
	"log"
	"regexp"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ttodoshi/weweather-bobot/internal/core/domain"
	"github.com/ttodoshi/weweather-bobot/internal/core/ports"
	"gorm.io/gorm"
)

// Определение основной клавиатуры для бота
var mainKeyboard = tg.NewReplyKeyboard(
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Погода"),
	),
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Добавить город"),
		tg.NewKeyboardButton("Удалить город"),
	),
	tg.NewKeyboardButtonRow(
		tg.NewKeyboardButton("Добавить оповещение"),
		tg.NewKeyboardButton("Удалить оповещение"),
	),
)

// Определение структуры CommandHandler
type CommandHandler struct {
	bot             *tg.BotAPI
	db              *gorm.DB
	weatherProvider ports.WeatherProvider
}

// Определение конструктора CommandHandler
func NewCommandHander(
	bot *tg.BotAPI,
	db *gorm.DB,
	weatherProvider ports.WeatherProvider,
) *CommandHandler {
	return &CommandHandler{
		bot, db, weatherProvider,
	}
}

// Обработка пришедшей команды
func (h *CommandHandler) HandleCommand(update tg.Update) {
	var msg tg.MessageConfig

	// Если обновление содержит текст
	if update.Message != nil {
		// Создаем новое сообщение для ответа
		msg = tg.NewMessage(update.Message.Chat.ID, "")

		// Если сообщение является командой
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				// Возращаем клавиатуру и сообщение
				msg.ReplyMarkup = mainKeyboard
				msg.Text = "Привет, Я телеграм-бот для погоды. Нажми на кнопку интересующей тебя команды ниже:"
			case "add":
				// Добавление города в базу данных после проверки, что пришло не пустое сообщение
				if city := strings.TrimSpace(update.Message.CommandArguments()); len(city) > 0 {
					msg.Text = "Добавлен город " + city
					h.db.Create(&domain.City{
						UserID: update.Message.From.ID,
						City:   city,
					})
				} else {
					msg.Text = "Нельзя добавить город без названия"
				}
			default:
				msg.Text = "Неизвестная команда"
			}
		} else {
			// Обработка не командного сообщения
			switch update.Message.Text {
			case "Погода":
				// Получение списка городов для пользователя
				var cities []string
				h.db.
					Select("city").
					Where("user_id = ?", update.Message.From.ID).
					Table("cities").
					Find(&cities)

				// Возращение погоды для городов пользователя
				if len(cities) > 0 {
					msg.Text = "Погода в ваших городах:\n\n"
					for _, w := range h.weatherProvider.FetchWeather(cities) {
						msg.Text += w
					}
				} else {
					msg.Text = "У вас ещё нет отслеживаемых городов"
				}
			case "Добавить город":
				msg.Text = "Для добавления города напишите /add <Название города>"
			case "Удалить город":
				// Получение списка городов для пользователя
				var cities []string
				h.db.
					Select("city").
					Where("user_id = ?", update.Message.From.ID).
					Table("cities").
					Find(&cities)

				// Создание inline-клавиатуры для выбора города, который нужно удалить, если у пользователя есть добавленные города
				if len(cities) > 0 {
					msg.Text = "Выберите город для удаления:"

					citiesKeyboard := tg.NewInlineKeyboardMarkup()
					for _, city := range cities {
						citiesKeyboard.InlineKeyboard = append(
							citiesKeyboard.InlineKeyboard,
							tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(city, city)),
						)
					}
					msg.ReplyMarkup = citiesKeyboard
				} else {
					msg.Text = "У вас ещё нет городов"
				}
			case "Добавить оповещение":
				msg.Text = "Напишите время в которое вы хотите получать оповещения. Например, 08:00"
			case "Удалить оповещение":
				// Получение списка оповещений для пользователя
				var notifications []string
				h.db.
					Select("time").
					Where("chat_id = ?", update.Message.From.ID).
					Table("notifications").
					Find(&notifications)

				// Создание inline-клавиатуры для выбора оповещения, которое нужно удалить, если у пользователя есть добавленные оповещения
				if len(notifications) > 0 {
					msg.Text = "Выберите оповещение для удаления:"

					notificationsKeyboard := tg.NewInlineKeyboardMarkup()
					for _, notification := range notifications {
						notificationsKeyboard.InlineKeyboard = append(
							notificationsKeyboard.InlineKeyboard,
							tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(notification, notification)),
						)
					}
					msg.ReplyMarkup = notificationsKeyboard
				} else {
					msg.Text = "У вас ещё нет установленных оповещений"
				}
			default:
				// Проверка, является ли сообщение валидным временем оповещения
				matched, err := regexp.MatchString("^(0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]$", update.Message.Text)
				if err != nil || !matched {
					msg.Text = "Неизвестная команда"
				} else {
					// Если время валидное, то сохраняем оповещение
					msg.Text = "Добавлено оповещение"
					h.db.Create(&domain.Notification{
						ChatID: update.Message.Chat.ID,
						UserID: update.Message.From.ID,
						Time:   update.Message.Text,
					})
				}
			}
		}
	} else if update.CallbackQuery != nil {
		// Обработка нажатий на inline-клавиатуру
		callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := h.bot.Request(callback); err != nil {
			panic(err)
		}

		msg = tg.NewMessage(update.CallbackQuery.Message.Chat.ID, "")

		switch update.CallbackQuery.Message.Text {
		// Если сообщение про удаление города, то удаляем город
		case "Выберите город для удаления:":
			msg.Text = "Удален город " + update.CallbackQuery.Data
			h.db.Delete(&domain.City{}, "user_id = ? AND city = ?", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
		// Если сообщение про удаление оповещения, то удаляем оповещение
		case "Выберите оповещение для удаления:":
			msg.Text = "Удалено оповещение в " + update.CallbackQuery.Data
			h.db.Delete(&domain.Notification{}, "chat_id = ? AND time = ?", update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
		}

		// Удаляем сообщение с клавиатурой
		deleteMessage := tg.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)
		if _, err := h.bot.Request(deleteMessage); err != nil {
			log.Panic(err)
		}
	}
	// Отправляем сообщение
	if _, err := h.bot.Send(msg); err != nil {
		log.Panic(err)
	}
}
