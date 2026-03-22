package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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

// sanitizeForLog маскирует потенциально чувствительные данные в логах
func sanitizeForLog(text string) string {
	if len(text) > 50 {
		return text[:50] + "..."
	}
	// Маскируем потенциальные токены (длинные строки с двоеточием)
	if len(text) > 30 && strings.Contains(text, ":") {
		return "[REDACTED]"
	}
	return text
}

func loadEnv() error {
	exePath, err := os.Executable()
	if err == nil {
		envPath := filepath.Join(filepath.Dir(exePath), ".env")
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("[INFO] Loaded .env from: %s", envPath)
			return nil
		}
	}

	if err := godotenv.Load(".env"); err == nil {
		log.Printf("[INFO] Loaded .env from current directory")
		return nil
	}

	if err := godotenv.Load("../../.env"); err == nil {
		log.Printf("[INFO] Loaded .env from ../../.env")
		return nil
	}

	return fmt.Errorf(".env file not found")
}

func main() {
	if err := loadEnv(); err != nil {
		log.Printf("[WARN] %v", err)
		log.Printf("[INFO] Continuing with environment variables only")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Printf("[ERROR] Token not found in TELEGRAM_BOT_TOKEN env variable")
		log.Printf("[HINT] Create .env file with TELEGRAM_BOT_TOKEN=your_token")
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
	updateConfig.Timeout = int(consts.PollingTimeout.Seconds())
	updates := bot.GetUpdatesChan(updateConfig)

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запуск обработки сообщений в горутине
	go func() {
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

			log.Printf("[INFO] [CHAT_ID:%d] Received message: %s", chatID, sanitizeForLog(text))

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
	}()

	// Ждём сигнал завершения
	sig := <-sigChan
	log.Printf("[INFO] Received signal %v, shutting down...", sig)

	// Закрываем канал обновлений
	bot.StopReceivingUpdates()

	// Даём время на завершение текущих операций
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	<-ctx.Done()
	log.Printf("[INFO] Bot stopped")
}
