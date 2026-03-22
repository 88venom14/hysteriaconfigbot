package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"hysconfigbot/internal/handlers"
	"hysconfigbot/pkg/client"
	"hysconfigbot/pkg/consts"
	"hysconfigbot/pkg/models"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func loadEnv() {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[WARN] Failed to get executable path: %v", err)
	} else {
		envPath := filepath.Join(filepath.Dir(exePath), ".env")
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("[INFO] Loaded .env from: %s", envPath)
			return
		}
	}

	if err := godotenv.Load(".env"); err == nil {
		log.Printf("[INFO] Loaded .env from current directory")
		return
	}

	if err := godotenv.Load("../../.env"); err == nil {
		log.Printf("[INFO] Loaded .env from ../../.env")
		return
	}

	log.Printf("[WARN] .env file not found, using environment variables")
}

func main() {
	loadEnv()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Printf("[ERROR] Token not found in TELEGRAM_BOT_TOKEN env variable")
		log.Fatalf("[FATAL] Bot initialization failed: missing token")
	}

	httpClient := client.NewHTTPClient()

	bot, err := tgbotapi.NewBotAPIWithClient(token, "https://api.telegram.org/bot%s/%s", httpClient)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to Telegram API: %v", err)
	}

	commands := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Запустить бота"},
		tgbotapi.BotCommand{Command: "goconfig", Description: "Создать YAML-конфиг Hysteria2"},
		tgbotapi.BotCommand{Command: "config", Description: "Показать мои конфиги"},
		tgbotapi.BotCommand{Command: "help", Description: "Показать справку"},
		tgbotapi.BotCommand{Command: "stop", Description: "Отменить создание конфига"},
	)
	if _, err := bot.Request(commands); err != nil {
		log.Printf("[WARN] Failed to set bot commands: %v", err)
	}

	log.Printf("[INFO] Bot started: %s", bot.Self.UserName)

	botState := models.NewBotState()
	handler := handlers.NewHandler(bot, botState)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = consts.PollingTimeout
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.CallbackQuery != nil {
			handler.HandleCallback(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		log.Printf("[INFO] [CHAT_ID:%d] Received message: %s", chatID, text)

		// Обработка команды /stop на любом этапе
		if text == "/stop" {
			handler.HandleStop(chatID)
			continue
		}

		step := botState.GetConfigStep(chatID)
		switch step {
		case models.StepWaitingServer:
			server := strings.TrimSpace(text)
			if server == "" {
				msg := tgbotapi.NewMessage(chatID, "Адрес сервера не может быть пустым. Введите адрес сервера:")
				bot.Send(msg)
				continue
			}
			handler.HandleServerAddress(chatID, server)
			continue
		case models.StepWaitingName:
			userName := strings.TrimSpace(text)
			if userName == "" {
				msg := tgbotapi.NewMessage(chatID, "Имя не может быть пустым. Введите имя пользователя:")
				bot.Send(msg)
				continue
			}
			handler.HandleUserName(chatID, userName)
			continue
		case models.StepWaitingSpeed:
			// Если пользователь ввёл текст вместо нажатия кнопки
			msg := tgbotapi.NewMessage(chatID, "❌ Пожалуйста, выберите скорость, нажав на одну из кнопок ниже:")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🤖 Авто", "speed_auto"),
					tgbotapi.NewInlineKeyboardButtonData("🔧 Свой вариант", "speed_custom"),
				),
			)
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
			continue
		case models.StepWaitingCustomSpeed:
			customSpeed := strings.TrimSpace(text)
			handler.HandleCustomSpeed(chatID, customSpeed)
			continue
		}

		switch {
		case text == "/start":
			handler.HandleStart(chatID)
		case text == "/help":
			handler.HandleHelp(chatID)
		case text == "/config":
			handler.HandleConfig(chatID)
		case text == "/goconfig":
			handler.HandleGoConfig(chatID)
		default:
			msg := tgbotapi.NewMessage(chatID, "Используйте /goconfig для создания конфига")
			bot.Send(msg)
		}
	}
}
