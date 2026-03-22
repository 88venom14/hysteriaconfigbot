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
	"hysconfigbot/pkg/generator"
	"hysconfigbot/pkg/models"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// loadEnv загружает переменные окружения из .env файла
func loadEnv() {
	// Пытаемся загрузить .env из директории исполняемого файла
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

	// Фолбэк 1: пытаемся загрузить из текущей рабочей директории
	if err := godotenv.Load(".env"); err == nil {
		log.Printf("[INFO] Loaded .env from current directory")
		return
	}

	// Фолбэк 2: пытаемся загрузить из родительской директории (для go run из cmd/bot)
	if err := godotenv.Load("../../.env"); err == nil {
		log.Printf("[INFO] Loaded .env from ../../.env")
		return
	}

	log.Printf("[WARN] .env file not found, using environment variables")
}

func main() {
	// Загрузка переменных окружения
	loadEnv()

	// Проверка токена
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Printf("[ERROR] Token not found in TELEGRAM_BOT_TOKEN env variable")
		log.Fatalf("[FATAL] Bot initialization failed: missing token")
	}

	// Создание HTTP-клиента с поддержкой прокси
	httpClient := client.NewHTTPClient()

	// Подключение к Telegram API
	bot, err := tgbotapi.NewBotAPIWithClient(token, "https://api.telegram.org/bot%s/%s", httpClient)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to Telegram API: %v", err)
	}

	// Установка команд бота
	commands := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Запустить бота"},
		tgbotapi.BotCommand{Command: "goconfig", Description: "Создать YAML-конфиг Hysteria2"},
		tgbotapi.BotCommand{Command: "config", Description: "Показать мои конфиги"},
		tgbotapi.BotCommand{Command: "help", Description: "Показать справку"},
	)
	if _, err := bot.Request(commands); err != nil {
		log.Printf("[WARN] Failed to set bot commands: %v", err)
	}

	log.Printf("[INFO] Bot started: %s", bot.Self.UserName)

	// Инициализация состояния и обработчика
	botState := models.NewBotState()
	handler := handlers.NewHandler(bot, botState)

	// Настройка получения обновлений
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = consts.PollingTimeout
	updates := bot.GetUpdatesChan(updateConfig)

	// Основной цикл обработки событий
	for update := range updates {
		// Обработка callback query (нажатие кнопок)
		if update.CallbackQuery != nil {
			handler.HandleCallback(update.CallbackQuery)
			continue
		}

		// Пропуск пустых сообщений
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		log.Printf("[INFO] [CHAT_ID:%d] Received message: %s", chatID, text)

		// Проверка шага создания конфига (приоритет)
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
			if !generator.IsValidLatinName(userName) {
				msg := tgbotapi.NewMessage(chatID, "❌ Имя должно содержать только латинские буквы (a-z, A-Z) и цифры (0-9), макс. 32 символа.\n\nПожалуйста, введите имя повторно:")
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)
				continue
			}
			handler.HandleUserName(chatID, userName)
			continue
		}

		// Обработка команд с эмодзи (от кнопок)
		cleanText := strings.TrimSpace(text)
		cleanText = strings.TrimPrefix(cleanText, "⚙️ ")
		cleanText = strings.TrimPrefix(cleanText, "🔑 ")
		cleanText = strings.TrimPrefix(cleanText, "📁 ")
		cleanText = strings.TrimPrefix(cleanText, "❓ ")

		// Обработка команд
		switch {
		case text == "/start" || cleanText == "/start":
			handler.HandleStart(chatID)
		case text == "/help" || cleanText == "/help":
			handler.HandleHelp(chatID)
		case text == "/config" || cleanText == "/config":
			handler.HandleConfig(chatID)
		case text == "/goconfig" || cleanText == "/goconfig":
			handler.HandleGoConfig(chatID)
		default:
			msg := tgbotapi.NewMessage(chatID, "Используйте /goconfig для создания конфига")
			bot.Send(msg)
		}
	}
}
