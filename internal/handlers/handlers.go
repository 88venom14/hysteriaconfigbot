package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"hysconfigbot/pkg/consts"
	"hysconfigbot/pkg/generator"
	"hysconfigbot/pkg/models"
)

type Handler struct {
	Bot   *tgbotapi.BotAPI
	State *models.BotState
}

func NewHandler(bot *tgbotapi.BotAPI, state *models.BotState) *Handler {
	return &Handler{
		Bot:   bot,
		State: state,
	}
}

func (h *Handler) HandleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, consts.BotWelcomeMessage)
	msg.ParseMode = tgbotapi.ModeMarkdown

	// Создаём inline-кнопки под сообщением
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Создать конфиг", "btn_goconfig"),
			tgbotapi.NewInlineKeyboardButtonData("📁 Мои конфиги", "btn_config"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Справка", "btn_help"),
		),
	)
	msg.ReplyMarkup = keyboard

	h.send(msg)
}

func (h *Handler) HandleHelp(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, consts.BotHelpMessage)
	msg.ParseMode = tgbotapi.ModeMarkdown

	// Создаём inline-кнопки под сообщением
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Создать конфиг", "btn_goconfig"),
			tgbotapi.NewInlineKeyboardButtonData("📁 Мои конфиги", "btn_config"),
		),
	)
	msg.ReplyMarkup = keyboard

	h.send(msg)
}

func (h *Handler) HandleStop(chatID int64) {
	// Очищаем состояние пользователя
	h.State.ClearUserConfigState(chatID)

	msg := tgbotapi.NewMessage(chatID, consts.BotStopMessage)
	msg.ParseMode = tgbotapi.ModeMarkdown

	// Убираем клавиатуру
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

	h.send(msg)
}

func (h *Handler) HandleGoConfig(chatID int64) {
	h.State.SetConfigStep(chatID, models.StepWaitingServer)
	msg := tgbotapi.NewMessage(chatID, consts.BotRequestServerMsg)
	if _, err := h.Bot.Send(msg); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send message: %v", chatID, err)
		h.State.SetConfigStep(chatID, models.StepNone)
	}
}

func (h *Handler) HandleServerAddress(chatID int64, server string) {
	if !generator.IsValidServerAddress(server) {
		msg := tgbotapi.NewMessage(chatID, consts.BotInvalidServerMsg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		h.send(msg)
		return
	}

	h.State.SetUserServer(chatID, server)
	h.State.SetConfigStep(chatID, models.StepWaitingName)
	msg := tgbotapi.NewMessage(chatID, consts.BotRequestNameMsg)
	if _, err := h.Bot.Send(msg); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send message: %v", chatID, err)
		h.State.SetConfigStep(chatID, models.StepNone)
	}
}

func (h *Handler) HandleUserName(chatID int64, userName string) {
	if !generator.IsValidLatinName(userName) {
		msg := tgbotapi.NewMessage(chatID, consts.BotInvalidNameMsg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		h.send(msg)
		return
	}

	h.State.SetUserName(chatID, userName)
	h.State.SetConfigStep(chatID, models.StepWaitingSpeed)

	msg := tgbotapi.NewMessage(chatID, consts.BotRequestSpeedMsg)

	// Создаём кнопки для выбора скорости
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Авто", "speed_auto"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 Свой вариант", "speed_custom"),
		),
	)
	msg.ReplyMarkup = keyboard

	if _, err := h.Bot.Send(msg); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send message: %v", chatID, err)
		h.State.SetConfigStep(chatID, models.StepNone)
	}
}

func (h *Handler) HandleCustomSpeed(chatID int64, customSpeed string) {
	// Парсим скорость в формате "UP DOWN"
	parts := strings.Fields(customSpeed)
	if len(parts) != 2 {
		msg := tgbotapi.NewMessage(chatID, consts.BotInvalidCustomSpeedMsg)
		h.send(msg)
		// Возвращаем к выбору скорости
		h.State.SetConfigStep(chatID, models.StepWaitingSpeed)
		return
	}

	up, errUp := strconv.Atoi(parts[0])
	down, errDown := strconv.Atoi(parts[1])

	if errUp != nil || errDown != nil || up < 0 || down < 0 {
		msg := tgbotapi.NewMessage(chatID, consts.BotInvalidCustomSpeedMsg)
		h.send(msg)
		// Возвращаем к выбору скорости
		h.State.SetConfigStep(chatID, models.StepWaitingSpeed)
		return
	}

	// Получаем данные из состояния
	server, serverExists := h.State.GetUserServer(chatID)
	if !serverExists {
		log.Printf("[ERROR] [CHAT_ID:%d] Server address not found", chatID)
		h.sendErrorMessage(chatID)
		return
	}

	userName, nameExists := h.State.GetUserName(chatID)
	if !nameExists {
		log.Printf("[ERROR] [CHAT_ID:%d] User name not found", chatID)
		h.sendErrorMessage(chatID)
		return
	}

	h.State.SetUserSpeed(chatID, up, down)

	password, err := generator.GeneratePassword()
	if err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to generate password: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	h.generateAndSendConfig(chatID, userName, password, server, up, down)
}

func (h *Handler) generateAndSendConfig(chatID int64, userName, password, server string, up, down int) {
	defer func() {
		h.State.ClearUserConfigState(chatID)
	}()

	config, err := generator.GenerateConfig(userName, password, server, up, down)
	if err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to generate config: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	credentialsMsg := fmt.Sprintf("%s: %s", userName, password)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s\n\n%s", consts.BotConfigCreatedMsg, credentialsMsg))
	if _, err := h.Bot.Send(msg); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send credentials: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	configCodeMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("```yaml\n%s\n```", config))
	configCodeMsg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := h.Bot.Send(configCodeMsg); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send config as code block: %v", chatID, err)
	}

	if err := h.sendConfigFile(chatID, userName, config); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send config file: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	configData := models.ConfigData{
		Name:     userName,
		Password: password,
		Config:   config,
		Server:   server,
		Up:       up,
		Down:     down,
	}
	if err := h.State.AddConfig(chatID, configData); err != nil {
		if err == models.ErrConfigLimitExceeded {
			msg := tgbotapi.NewMessage(chatID, "❌ Достигнут лимит конфигов (10).\n\nУдалите старые конфиги через /clear или создайте нового бота.")
			msg.ParseMode = tgbotapi.ModeMarkdown
			h.send(msg)
			return
		}
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to add config: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	h.sendRetryButton(chatID)

	log.Printf("[INFO] [CHAT_ID:%d] Config sent to user: %s (server: %s, up: %d, down: %d)", chatID, userName, server, up, down)
}

func (h *Handler) sendInvalidSpeedMessage(chatID int64) {
	h.sendError(chatID, consts.BotInvalidSpeedMsg)
}

func (h *Handler) HandleSpeedAuto(chatID int64) {
	// Получаем данные из состояния
	server, serverExists := h.State.GetUserServer(chatID)
	if !serverExists {
		log.Printf("[ERROR] [CHAT_ID:%d] Server address not found", chatID)
		h.sendErrorMessage(chatID)
		return
	}

	userName, nameExists := h.State.GetUserName(chatID)
	if !nameExists {
		log.Printf("[ERROR] [CHAT_ID:%d] User name not found", chatID)
		h.sendErrorMessage(chatID)
		return
	}

	// Автоматический режим (0, 0)
	h.State.SetUserSpeed(chatID, 0, 0)

	password, err := generator.GeneratePassword()
	if err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to generate password: %v", chatID, err)
		h.sendErrorMessage(chatID)
		return
	}

	h.generateAndSendConfig(chatID, userName, password, server, 0, 0)
}

func (h *Handler) HandleSpeedCustom(chatID int64) {
	h.State.SetConfigStep(chatID, models.StepWaitingCustomSpeed)
	msg := tgbotapi.NewMessage(chatID, consts.BotRequestCustomSpeedMsg)
	h.send(msg)
}

func (h *Handler) HandleConfig(chatID int64) {
	configs, exists := h.State.GetConfigs(chatID)
	if !exists || len(configs) == 0 {
		msg := tgbotapi.NewMessage(chatID, consts.BotNoConfigsMessage)
		msg.ParseMode = tgbotapi.ModeMarkdown
		h.send(msg)
		return
	}

	var configList strings.Builder
	configList.WriteString(consts.BotConfigsListMessage)

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	for i, cfg := range configs {
		configList.WriteString(fmt.Sprintf("🔹 **%s**: `%s`\n", cfg.Name, cfg.Password))
		configList.WriteString(fmt.Sprintf("   🌐 Сервер: `%s`\n\n", cfg.Server))

		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("📥 %s_config.yaml", cfg.Name),
			fmt.Sprintf("download_%d", i),
		)
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
	}

	if len(configs) > 0 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Очистить историю", "clear_confirm"),
		))
	}

	msg := tgbotapi.NewMessage(chatID, configList.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	h.send(msg)
}

func (h *Handler) HandleClearConfirm(chatID int64) {
	count := h.State.GetConfigsCount(chatID)
	if count == 0 {
		msg := tgbotapi.NewMessage(chatID, consts.BotNoConfigsMessage)
		h.send(msg)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(consts.BotClearButton, "clear_execute"),
			tgbotapi.NewInlineKeyboardButtonData(consts.BotCancelButton, "clear_cancel"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(consts.BotClearConfirmMessage, count))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	h.send(msg)
}

func (h *Handler) HandleClearExecute(chatID int64) {
	h.State.ClearConfigs(chatID)
	msg := tgbotapi.NewMessage(chatID, consts.BotClearDoneMessage)
	h.send(msg)
	log.Printf("[INFO] [CHAT_ID:%d] Configs cleared", chatID)
}

func (h *Handler) HandleDownload(chatID int64, configIndex int) {
	if configIndex < 0 {
		log.Printf("[ERROR] [CHAT_ID:%d] Invalid config index: %d (negative)", chatID, configIndex)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка: неверный индекс конфига")
		h.send(msg)
		return
	}

	// Получаем КОПИЮ конфига — данные безопасны после выхода из функции
	cfg, exists := h.State.GetConfigByIndexCopy(chatID, configIndex)
	if !exists {
		log.Printf("[ERROR] [CHAT_ID:%d] Invalid config index: %d (not found)", chatID, configIndex)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка: конфиг не найден или у вас нет доступа к нему")
		h.send(msg)
		return
	}

	// Теперь можно безопасно использовать cfg, даже если оригинал будет удалён
	if err := h.sendConfigFile(chatID, cfg.Name, cfg.Config); err != nil {
		log.Printf("[ERROR] [CHAT_ID:%d] Failed to send config file: %v", chatID, err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при отправке файла")
		h.send(msg)
		return
	}

	log.Printf("[INFO] [CHAT_ID:%d] Config downloaded: %s", chatID, cfg.Name)
}

func (h *Handler) HandleCallback(callbackQuery *tgbotapi.CallbackQuery) {
	chatID := callbackQuery.Message.Chat.ID

	switch {
	case callbackQuery.Data == "btn_goconfig":
		h.HandleGoConfig(chatID)
		h.answerCallback(callbackQuery.ID, "")

	case callbackQuery.Data == "btn_config":
		h.HandleConfig(chatID)
		h.answerCallback(callbackQuery.ID, "")

	case callbackQuery.Data == "btn_help":
		h.HandleHelp(chatID)
		h.answerCallback(callbackQuery.ID, "")

	case callbackQuery.Data == "retry":
		h.HandleGoConfig(chatID)
		h.answerCallback(callbackQuery.ID, "")

	case callbackQuery.Data == "speed_auto":
		h.HandleSpeedAuto(chatID)
		h.answerCallback(callbackQuery.ID, "Выбран автоматический режим")

	case callbackQuery.Data == "speed_custom":
		h.HandleSpeedCustom(chatID)
		h.answerCallback(callbackQuery.ID, "Введите вашу скорость")

	case strings.HasPrefix(callbackQuery.Data, "download_"):
		var index int
		if _, err := fmt.Sscanf(callbackQuery.Data, "download_%d", &index); err != nil {
			log.Printf("[ERROR] [CHAT_ID:%d] Invalid callback data: %s", chatID, callbackQuery.Data)
			h.answerCallback(callbackQuery.ID, "Ошибка формата")
			return
		}
		h.HandleDownload(chatID, index)
		h.answerCallback(callbackQuery.ID, "Отправляю конфиг...")

	case callbackQuery.Data == "clear_confirm":
		h.HandleClearConfirm(chatID)
		h.answerCallback(callbackQuery.ID, "")

	case callbackQuery.Data == "clear_execute":
		h.HandleClearExecute(chatID)
		h.answerCallback(callbackQuery.ID, "Конфиги удалены")

	case callbackQuery.Data == "clear_cancel":
		h.answerCallback(callbackQuery.ID, "Отменено")
	}
}

func (h *Handler) send(msg tgbotapi.Chattable) {
	result, err := h.Bot.Send(msg)
	if err != nil {
		log.Printf("[ERROR] Failed to send message: %v", err)
	} else {
		log.Printf("[INFO] [CHAT_ID:%d] Message sent", result.Chat.ID)
	}
}

func (h *Handler) sendErrorMessage(chatID int64) {
	h.sendError(chatID, consts.BotErrorGenericMsg)
}

func (h *Handler) sendConfigFile(chatID int64, userName, config string) error {
	// Санитизируем имя файла перед использованием
	safeName := generator.SanitizeFileName(userName)
	fileName := fmt.Sprintf("%s_config.yaml", safeName)
	tmpDir := os.TempDir()
	tmpFilePath := filepath.Join(tmpDir, fileName)

	// Дополнительно проверяем, что путь остаётся внутри temp-директории
	cleanPath := filepath.Clean(tmpFilePath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(tmpDir)) {
		return fmt.Errorf("invalid file path: %s", tmpFilePath)
	}

	if err := os.WriteFile(tmpFilePath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer os.Remove(tmpFilePath)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(tmpFilePath))
	_, err := h.Bot.Send(doc)
	if err != nil {
		return fmt.Errorf("failed to send document: %w", err)
	}

	return nil
}

// sendError отправляет сообщение об ошибке пользователю
func (h *Handler) sendError(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	h.send(msg)
}

// sendAndLogError отправляет ошибку и логирует её
func (h *Handler) sendAndLogError(chatID int64, logMsg string, userMsg string) {
	log.Print(logMsg)
	h.sendError(chatID, userMsg)
}

func (h *Handler) sendRetryButton(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(consts.BotRetryButton, "retry"),
			tgbotapi.NewInlineKeyboardButtonData("❗ Перейти к командам", "btn_help"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Хотите сгенерировать ещё один конфиг?")
	msg.ReplyMarkup = keyboard
	h.send(msg)
}

func (h *Handler) answerCallback(callbackID, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.Bot.Request(callback); err != nil {
		log.Printf("[ERROR] Failed to answer callback %s: %v", callbackID, err)
	}
}
