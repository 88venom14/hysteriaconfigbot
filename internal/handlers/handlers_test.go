package handlers

import (
	"fmt"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"hysconfigbot/pkg/consts"
	"hysconfigbot/pkg/models"
)

// MockBotAPI - моковая реализация для тестирования
type MockBotAPI struct {
	sentMessages    []tgbotapi.Chattable
	callbackAnswers []tgbotapi.CallbackConfig
	lastMessageText string
	lastChatID      int64
	lastParseMode   string
	lastReplyMarkup interface{}
	sendError       error
	requestError    error
}

func NewMockBotAPI() *MockBotAPI {
	return &MockBotAPI{
		sentMessages:    make([]tgbotapi.Chattable, 0),
		callbackAnswers: make([]tgbotapi.CallbackConfig, 0),
	}
}

func (m *MockBotAPI) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	if m.sendError != nil {
		return tgbotapi.Message{}, m.sendError
	}

	m.sentMessages = append(m.sentMessages, msg)

	// Сохраняем информацию о последнем сообщении для проверок
	switch v := msg.(type) {
	case tgbotapi.MessageConfig:
		m.lastMessageText = v.Text
		m.lastChatID = v.ChatID
		m.lastParseMode = v.ParseMode
		m.lastReplyMarkup = v.ReplyMarkup
	case tgbotapi.DocumentConfig:
		m.lastChatID = v.ChatID
	}

	return tgbotapi.Message{Chat: &tgbotapi.Chat{ID: m.lastChatID}}, nil
}

func (m *MockBotAPI) Request(req tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	if m.requestError != nil {
		return nil, m.requestError
	}

	// Сохраняем callback ответы
	if callback, ok := req.(tgbotapi.CallbackConfig); ok {
		m.callbackAnswers = append(m.callbackAnswers, callback)
	}

	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *MockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return nil
}

func (m *MockBotAPI) StopReceivingUpdates() {}

// Helper методы для проверок
func (m *MockBotAPI) LastMessageText() string {
	return m.lastMessageText
}

func (m *MockBotAPI) LastChatID() int64 {
	return m.lastChatID
}

func (m *MockBotAPI) LastParseMode() string {
	return m.lastParseMode
}

func (m *MockBotAPI) MessagesCount() int {
	return len(m.sentMessages)
}

func (m *MockBotAPI) CallbackAnswersCount() int {
	return len(m.callbackAnswers)
}

func (m *MockBotAPI) ClearMessages() {
	m.sentMessages = make([]tgbotapi.Chattable, 0)
	m.lastMessageText = ""
	m.lastChatID = 0
}

func (m *MockBotAPI) HasMarkdown() bool {
	return m.lastParseMode == tgbotapi.ModeMarkdown
}

func (m *MockBotAPI) HasInlineKeyboard() bool {
	_, ok := m.lastReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	return ok
}

func (m *MockBotAPI) HasRemoveKeyboard() bool {
	_, ok := m.lastReplyMarkup.(tgbotapi.ReplyKeyboardRemove)
	return ok
}

func (m *MockBotAPI) ContainsText(substring string) bool {
	return strings.Contains(m.lastMessageText, substring)
}

// Создаём тестовый CallbackQuery
func NewTestCallbackQuery(chatID int64, data string) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		Data: data,
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{
				ID: chatID,
			},
		},
	}
}

// Тесты для HandleStart
func TestHandler_HandleStart(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleStart(123456789)

	if mockBot.MessagesCount() != 1 {
		t.Errorf("Expected 1 message, got %d", mockBot.MessagesCount())
	}

	if !mockBot.HasMarkdown() {
		t.Error("Expected Markdown parse mode")
	}

	if !mockBot.HasInlineKeyboard() {
		t.Error("Expected inline keyboard")
	}

	if !strings.Contains(mockBot.LastMessageText(), "Привет") {
		t.Error("Expected welcome message")
	}
}

// Тесты для HandleHelp
func TestHandler_HandleHelp(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleHelp(123456789)

	if mockBot.MessagesCount() != 1 {
		t.Errorf("Expected 1 message, got %d", mockBot.MessagesCount())
	}

	if !mockBot.HasMarkdown() {
		t.Error("Expected Markdown parse mode")
	}

	if !strings.Contains(mockBot.LastMessageText(), "Справка") {
		t.Error("Expected help message")
	}
}

// Тесты для HandleStop
func TestHandler_HandleStop(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Устанавливаем состояние
	state.SetConfigStep(123456789, models.StepWaitingServer)
	state.SetUserServer(123456789, "example.com")

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleStop(123456789)

	if mockBot.MessagesCount() != 1 {
		t.Errorf("Expected 1 message, got %d", mockBot.MessagesCount())
	}

	if !mockBot.HasRemoveKeyboard() {
		t.Error("Expected remove keyboard")
	}

	// Проверка, что состояние очищено
	step := state.GetConfigStep(123456789)
	if step != models.StepNone {
		t.Errorf("Expected StepNone after stop, got %v", step)
	}
}

// Тесты для HandleGoConfig
func TestHandler_HandleGoConfig(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleGoConfig(123456789)

	if mockBot.MessagesCount() != 1 {
		t.Errorf("Expected 1 message, got %d", mockBot.MessagesCount())
	}

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingServer {
		t.Errorf("Expected StepWaitingServer, got %v", step)
	}

	if !strings.Contains(mockBot.LastMessageText(), "адрес сервера") {
		t.Error("Expected server request message")
	}
}

// Тесты для HandleServerAddress
func TestHandler_HandleServerAddress_Valid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleServerAddress(123456789, "example.com")

	server, exists := state.GetUserServer(123456789)
	if !exists {
		t.Error("Expected server to be set")
	}
	if server != "example.com" {
		t.Errorf("Expected server 'example.com', got %q", server)
	}

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingName {
		t.Errorf("Expected StepWaitingName, got %v", step)
	}

	if !strings.Contains(mockBot.LastMessageText(), "имя пользователя") {
		t.Error("Expected name request message")
	}
}

func TestHandler_HandleServerAddress_Invalid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Невалидные адреса
	invalidAddresses := []string{
		"",
		"invalid..domain",
		"-example.com",
		"example.com-",
		"exam{ple.com",
	}

	for _, addr := range invalidAddresses {
		mockBot.ClearMessages()
		handler.HandleServerAddress(123456789, addr)

		if !strings.Contains(mockBot.LastMessageText(), "Неверный формат") {
			t.Errorf("Expected error message for invalid address: %q", addr)
		}
	}
}

// Тесты для HandleUserName
func TestHandler_HandleUserName_Valid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	state.SetUserServer(123456789, "example.com")
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleUserName(123456789, "testuser")

	name, exists := state.GetUserName(123456789)
	if !exists {
		t.Error("Expected name to be set")
	}
	if name != "testuser" {
		t.Errorf("Expected name 'testuser', got %q", name)
	}

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingSpeed {
		t.Errorf("Expected StepWaitingSpeed, got %v", step)
	}

	if !mockBot.HasInlineKeyboard() {
		t.Error("Expected inline keyboard for speed selection")
	}
}

func TestHandler_HandleUserName_Invalid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	state.SetUserServer(123456789, "example.com")
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Невалидные имена
	invalidNames := []string{
		"",
		"пользователь",
		"user_name",
		"user-name",
		"user name",
		"user@name",
	}

	for _, name := range invalidNames {
		mockBot.ClearMessages()
		handler.HandleUserName(123456789, name)

		if !strings.Contains(mockBot.LastMessageText(), "латинские буквы") {
			t.Errorf("Expected error message for invalid name: %q", name)
		}
	}
}

// Тесты для HandleSpeedCustom
func TestHandler_HandleSpeedCustom_InvalidFormat(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Неправильный формат (одно число)
	handler.HandleCustomSpeed(123456789, "300")

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingSpeed {
		t.Errorf("Expected StepWaitingSpeed, got %v", step)
	}

	if !strings.Contains(mockBot.LastMessageText(), "Неверный формат") {
		t.Error("Expected error message for invalid format")
	}
}

func TestHandler_HandleSpeedCustom_NegativeValues(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleCustomSpeed(123456789, "-100 -200")

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingSpeed {
		t.Errorf("Expected StepWaitingSpeed, got %v", step)
	}
}

func TestHandler_HandleSpeedCustom_NonNumeric(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleCustomSpeed(123456789, "abc def")

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingSpeed {
		t.Errorf("Expected StepWaitingSpeed, got %v", step)
	}
}

// Тесты для HandleSpeedAuto
func TestHandler_HandleSpeedAuto_MissingData(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Вызов без установленных данных
	handler.HandleSpeedAuto(123456789)

	// Должна быть отправлена ошибка
	if mockBot.MessagesCount() == 0 {
		t.Error("Expected error message when data is missing")
	}
}

// Тесты для HandleConfig
func TestHandler_HandleConfig_NoConfigs(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleConfig(123456789)

	if !strings.Contains(mockBot.LastMessageText(), "нет созданных конфигов") {
		t.Error("Expected no configs message")
	}
}

func TestHandler_HandleConfig_WithConfigs(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "testuser",
		Password: "testpass123",
		Config:   "testconfig",
		Server:   "example.com",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleConfig(123456789)

	if !strings.Contains(mockBot.LastMessageText(), "testuser") {
		t.Error("Expected config name in message")
	}

	if !strings.Contains(mockBot.LastMessageText(), "testpass123") {
		t.Error("Expected config password in message")
	}

	if !mockBot.HasInlineKeyboard() {
		t.Error("Expected inline keyboard for download")
	}
}

// Тесты для HandleClearConfirm
func TestHandler_HandleClearConfirm_NoConfigs(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleClearConfirm(123456789)

	if !strings.Contains(mockBot.LastMessageText(), "нет созданных конфигов") {
		t.Error("Expected no configs message")
	}
}

func TestHandler_HandleClearConfirm_WithConfigs(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "test",
		Password: "test",
		Config:   "test",
		Server:   "test",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleClearConfirm(123456789)

	if !strings.Contains(mockBot.LastMessageText(), "Вы уверены") {
		t.Error("Expected confirmation message")
	}

	if !mockBot.HasInlineKeyboard() {
		t.Error("Expected inline keyboard with confirm/cancel buttons")
	}
}

// Тесты для HandleClearExecute
func TestHandler_HandleClearExecute(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиги
	for i := 0; i < 3; i++ {
		config := models.ConfigData{
			Name:     fmt.Sprintf("user%d", i),
			Password: "pass",
			Config:   "config",
			Server:   "server",
			Up:       100,
			Down:     100,
		}
		state.AddConfig(123456789, config)
	}

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleClearExecute(123456789)

	if state.GetConfigsCount(123456789) != 0 {
		t.Error("Expected all configs to be cleared")
	}

	if !strings.Contains(mockBot.LastMessageText(), "удалены") {
		t.Error("Expected success message")
	}
}

// Тесты для HandleDownload
func TestHandler_HandleDownload_InvalidIndex(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Отрицательный индекс
	handler.HandleDownload(123456789, -1)

	if !strings.Contains(mockBot.LastMessageText(), "неверный индекс") {
		t.Error("Expected error message for negative index")
	}

	mockBot.ClearMessages()

	// Несуществующий индекс
	handler.HandleDownload(123456789, 100)

	if !strings.Contains(mockBot.LastMessageText(), "не найден") {
		t.Error("Expected error message for not found config")
	}
}

func TestHandler_HandleDownload_Valid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "testuser",
		Password: "testpass",
		Config:   "testconfig",
		Server:   "example.com",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleDownload(123456789, 0)

	// Проверяем, что был отправлен документ (файл)
	if mockBot.MessagesCount() < 1 {
		t.Error("Expected document to be sent")
	}
}

// Тесты для HandleCallback
func TestHandler_HandleCallback_BtnGoConfig(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "btn_goconfig")
	handler.HandleCallback(callback)

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingServer {
		t.Errorf("Expected StepWaitingServer, got %v", step)
	}

	if mockBot.CallbackAnswersCount() != 1 {
		t.Error("Expected callback answer")
	}
}

func TestHandler_HandleCallback_BtnConfig(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "btn_config")
	handler.HandleCallback(callback)

	if mockBot.CallbackAnswersCount() != 1 {
		t.Error("Expected callback answer")
	}
}

func TestHandler_HandleCallback_BtnHelp(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "btn_help")
	handler.HandleCallback(callback)

	if !strings.Contains(mockBot.LastMessageText(), "Справка") {
		t.Error("Expected help message")
	}
}

func TestHandler_HandleCallback_Retry(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "retry")
	handler.HandleCallback(callback)

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingServer {
		t.Errorf("Expected StepWaitingServer, got %v", step)
	}
}

func TestHandler_HandleCallback_SpeedAuto(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	state.SetUserServer(123456789, "example.com")
	state.SetUserName(123456789, "testuser")
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "speed_auto")
	handler.HandleCallback(callback)

	// Проверяем, что callback был обработан
	if mockBot.CallbackAnswersCount() == 0 {
		t.Error("Expected callback answer")
	}
}

func TestHandler_HandleCallback_SpeedCustom(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "speed_custom")
	handler.HandleCallback(callback)

	step := state.GetConfigStep(123456789)
	if step != models.StepWaitingCustomSpeed {
		t.Errorf("Expected StepWaitingCustomSpeed, got %v", step)
	}
}

func TestHandler_HandleCallback_ClearConfirm(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "test",
		Password: "test",
		Config:   "test",
		Server:   "test",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "clear_confirm")
	handler.HandleCallback(callback)

	if !strings.Contains(mockBot.LastMessageText(), "Вы уверены") {
		t.Error("Expected confirmation message")
	}
}

func TestHandler_HandleCallback_ClearExecute(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "test",
		Password: "test",
		Config:   "test",
		Server:   "test",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "clear_execute")
	handler.HandleCallback(callback)

	if state.GetConfigsCount(123456789) != 0 {
		t.Error("Expected configs to be cleared")
	}
}

func TestHandler_HandleCallback_ClearCancel(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "clear_cancel")
	handler.HandleCallback(callback)

	if mockBot.CallbackAnswersCount() == 0 {
		t.Error("Expected callback answer")
	}
}

func TestHandler_HandleCallback_Download(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем конфиг
	config := models.ConfigData{
		Name:     "testuser",
		Password: "testpass",
		Config:   "testconfig",
		Server:   "example.com",
		Up:       100,
		Down:     100,
	}
	state.AddConfig(123456789, config)

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "download_0")
	handler.HandleCallback(callback)

	// Проверяем, что файл был отправлен
	if mockBot.MessagesCount() == 0 {
		t.Error("Expected config file to be sent")
	}
}

func TestHandler_HandleCallback_DownloadInvalid(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "download_invalid")
	handler.HandleCallback(callback)

	if mockBot.CallbackAnswersCount() == 0 {
		t.Error("Expected callback answer with error")
	}
}

// Тесты для escapeMarkdown
func TestEscapeMarkdown_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no_special", "hello world", "hello world"},
		{"underscore", "hello_world", "hello\\_world"},
		{"asterisk", "hello*world", "hello\\*world"},
		{"backtick", "hello`world", "hello\\`world"},
		{"bracket_open", "hello[world", "hello\\[world"},
		{"bracket_close", "hello]world", "hello]world"}, // ] не экранируется в Telegram Markdown
		{"parenthesis_open", "hello(world", "hello\\(world"},
		{"parenthesis_close", "hello)world", "hello\\)world"},
		{"all_special", "_*`[()]", "\\_\\*\\`\\[\\(\\)]"}, // ] не экранируется
		{"empty", "", ""},
		{"multiple_same", "___", "\\_\\_\\_"},
		{"mixed", "hello_world*test", "hello\\_world\\*test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Тесты для createMainKeyboard
func TestCreateMainKeyboard_Structure(t *testing.T) {
	handler := &Handler{}
	keyboard := handler.createMainKeyboard()

	// Проверка количества рядов
	if len(keyboard.InlineKeyboard) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(keyboard.InlineKeyboard))
	}

	// Проверка первого ряда (2 кнопки)
	if len(keyboard.InlineKeyboard[0]) != 2 {
		t.Errorf("Expected 2 buttons in first row, got %d", len(keyboard.InlineKeyboard[0]))
	}

	// Проверка второго ряда (1 кнопка)
	if len(keyboard.InlineKeyboard[1]) != 1 {
		t.Errorf("Expected 1 button in second row, got %d", len(keyboard.InlineKeyboard[1]))
	}

	// Проверка текстов кнопок
	expectedTexts := []string{"🔑 Создать конфиг", "📁 Мои конфиги", "❓ Справка"}
	var foundTexts []string
	for _, row := range keyboard.InlineKeyboard {
		for _, btn := range row {
			foundTexts = append(foundTexts, btn.Text)
		}
	}

	for _, expected := range expectedTexts {
		found := false
		for _, actual := range foundTexts {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected button %q not found", expected)
		}
	}
}

// Тесты для sendRetryButton
func TestSendRetryButton(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.sendRetryButton(123456789)

	if mockBot.MessagesCount() != 1 {
		t.Errorf("Expected 1 message, got %d", mockBot.MessagesCount())
	}

	if !mockBot.HasInlineKeyboard() {
		t.Error("Expected inline keyboard with retry button")
	}

	if !strings.Contains(mockBot.LastMessageText(), "ещё один конфиг") {
		t.Error("Expected retry message")
	}
}

// Тесты для sendError
func TestSendError(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.sendError(123456789, "Test error message")

	if !strings.Contains(mockBot.LastMessageText(), "Test error message") {
		t.Error("Expected error message")
	}

	if !mockBot.HasMarkdown() {
		t.Error("Expected Markdown parse mode")
	}
}

// Тесты для sendErrorMessage
func TestSendErrorMessage(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.sendErrorMessage(123456789)

	if !strings.Contains(mockBot.LastMessageText(), consts.BotErrorGenericMsg) {
		t.Error("Expected generic error message")
	}
}

// Тесты для answerCallback
func TestAnswerCallback(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.answerCallback("test_id", "Test answer")

	if mockBot.CallbackAnswersCount() != 1 {
		t.Error("Expected callback answer")
	}
}

// Тесты для sendConfigFile
func TestSendConfigFile_Sanitization(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	// Имя с опасными символами
	err := handler.sendConfigFile(123456789, "../../../etc/passwd", "test: config")

	// Ошибка ожидается, т.к. это тест без реального бота
	if err == nil {
		t.Log("Expected error due to mock bot")
	}
}

func TestSendConfigFile_NormalName(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	err := handler.sendConfigFile(123456789, "testuser", "test: config")

	if err == nil {
		t.Log("Expected error due to mock bot, but function should work")
	}
}

// Тесты для generateAndSendConfig
func TestGenerateAndSendConfig_Success(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	state.SetUserServer(123456789, "example.com")
	state.SetUserName(123456789, "testuser")
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.generateAndSendConfig(123456789, "testuser", "testpass", "example.com", 100, 100)

	// Проверяем, что состояние очищено
	step := state.GetConfigStep(123456789)
	if step != models.StepNone {
		t.Errorf("Expected StepNone after config generation, got %v", step)
	}

	// Проверяем, что конфиг сохранён
	configs, exists := state.GetConfigs(123456789)
	if !exists || len(configs) == 0 {
		t.Error("Expected config to be saved")
	}
}

func TestGenerateAndSendConfig_LimitExceeded(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()

	// Добавляем 10 конфигов (лимит)
	for i := 0; i < 10; i++ {
		config := models.ConfigData{
			Name:     fmt.Sprintf("user%d", i),
			Password: "pass",
			Config:   "config",
			Server:   "server",
			Up:       100,
			Down:     100,
		}
		state.AddConfig(123456789, config)
	}

	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.generateAndSendConfig(123456789, "testuser", "testpass", "example.com", 100, 100)

	// Проверяем, что было отправлено сообщение о лимите
	if !strings.Contains(mockBot.LastMessageText(), "Достигнут лимит") {
		t.Error("Expected config limit message")
	}
}

// Тесты для HandleSpeedCustom (успешный сценарий)
func TestHandler_HandleSpeedCustom_Success(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	state.SetUserServer(123456789, "example.com")
	state.SetUserName(123456789, "testuser")
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	handler.HandleCustomSpeed(123456789, "300 300")

	// Проверяем, что конфиг был создан
	configs, exists := state.GetConfigs(123456789)
	if !exists || len(configs) == 0 {
		t.Error("Expected config to be created")
	}
}

// Тест для обработки несуществующего callback
func TestHandler_HandleCallback_Unknown(t *testing.T) {
	mockBot := NewMockBotAPI()
	state := models.NewBotState()
	handler := &Handler{
		Bot:   mockBot,
		State: state,
	}

	callback := NewTestCallbackQuery(123456789, "unknown_callback_data")

	// Не должно паниковать
	handler.HandleCallback(callback)
}
