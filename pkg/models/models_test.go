package models

import (
	"testing"
)

func TestIsValidChatID(t *testing.T) {
	tests := []struct {
		name     string
		chatID   int64
		expected bool
	}{
		{"valid_personal", 123456789, true},
		{"valid_group", -987654321, true},
		{"valid_channel", -1001234567890, true},
		{"zero", 0, false},
		{"small_positive", 100, true},
		{"small_negative", -100, true},
		{"valid_large", 9223372036854775807, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidChatID(tt.chatID)
			if result != tt.expected {
				t.Errorf("IsValidChatID(%d) = %v, want %v", tt.chatID, result, tt.expected)
			}
		})
	}
}

func TestBotState_BasicOperations(t *testing.T) {
	state := NewBotState()
	chatID := int64(123456789)

	// Проверка начального состояния
	if step := state.GetConfigStep(chatID); step != StepNone {
		t.Errorf("Initial step should be StepNone, got %v", step)
	}

	// Установка шага
	state.SetConfigStep(chatID, StepWaitingServer)
	if step := state.GetConfigStep(chatID); step != StepWaitingServer {
		t.Errorf("Expected StepWaitingServer, got %v", step)
	}

	// Установка сервера
	state.SetUserServer(chatID, "example.com")
	server, exists := state.GetUserServer(chatID)
	if !exists {
		t.Error("Server should exist")
	}
	if server != "example.com" {
		t.Errorf("Expected server 'example.com', got '%s'", server)
	}

	// Установка имени
	state.SetUserName(chatID, "testuser")
	name, exists := state.GetUserName(chatID)
	if !exists {
		t.Error("Name should exist")
	}
	if name != "testuser" {
		t.Errorf("Expected name 'testuser', got '%s'", name)
	}

	// Установка скорости
	state.SetUserSpeed(chatID, 300, 300)
	state.SetConfigStep(chatID, StepWaitingSpeed)

	// Очистка состояния
	state.ClearUserConfigState(chatID)
	if step := state.GetConfigStep(chatID); step != StepNone {
		t.Errorf("After clear, step should be StepNone, got %v", step)
	}
	_, exists = state.GetUserServer(chatID)
	if exists {
		t.Error("Server should not exist after clear")
	}
}

func TestBotState_AddConfig(t *testing.T) {
	state := NewBotState()
	chatID := int64(987654321)

	// Добавление конфигов
	for i := 0; i < 5; i++ {
		config := ConfigData{
			Name:     "user",
			Password: "pass",
			Config:   "config",
			Server:   "server",
			Up:       100,
			Down:     100,
		}
		if err := state.AddConfig(chatID, config); err != nil {
			t.Fatalf("AddConfig() error = %v", err)
		}
	}

	// Проверка количества
	count := state.GetConfigsCount(chatID)
	if count != 5 {
		t.Errorf("Expected 5 configs, got %d", count)
	}

	// Получение списка
	configs, exists := state.GetConfigs(chatID)
	if !exists {
		t.Error("Configs should exist")
	}
	if len(configs) != 5 {
		t.Errorf("Expected 5 configs in list, got %d", len(configs))
	}

	// Получение по индексу
	cfg, exists := state.GetConfigByIndex(chatID, 0)
	if !exists {
		t.Error("Config at index 0 should exist")
	}
	if cfg.Name != "user" {
		t.Errorf("Expected config name 'user', got '%s'", cfg.Name)
	}

	// Несуществующий индекс
	_, exists = state.GetConfigByIndex(chatID, 100)
	if exists {
		t.Error("Config at index 100 should not exist")
	}

	// Отрицательный индекс
	_, exists = state.GetConfigByIndex(chatID, -1)
	if exists {
		t.Error("Config at negative index should not exist")
	}
}

func TestBotState_ConfigLimit(t *testing.T) {
	state := NewBotState()
	chatID := int64(111222333)

	// Добавление 10 конфигов (лимит)
	for i := 0; i < 10; i++ {
		config := ConfigData{
			Name:     "user",
			Password: "pass",
			Config:   "config",
			Server:   "server",
			Up:       100,
			Down:     100,
		}
		if err := state.AddConfig(chatID, config); err != nil {
			t.Fatalf("AddConfig() iteration %d error = %v", i, err)
		}
	}

	// 11-й конфиг должен вернуть ошибку
	config := ConfigData{
		Name:     "overflow",
		Password: "overflow",
		Config:   "overflow",
		Server:   "overflow",
		Up:       100,
		Down:     100,
	}
	if err := state.AddConfig(chatID, config); err != ErrConfigLimitExceeded {
		t.Errorf("Expected ErrConfigLimitExceeded, got %v", err)
	}
}

func TestBotState_ClearConfigs(t *testing.T) {
	state := NewBotState()
	chatID := int64(555666777)

	// Добавление конфигов
	for i := 0; i < 3; i++ {
		config := ConfigData{
			Name:     "user",
			Password: "pass",
			Config:   "config",
			Server:   "server",
			Up:       100,
			Down:     100,
		}
		state.AddConfig(chatID, config)
	}

	// Проверка
	count := state.GetConfigsCount(chatID)
	if count != 3 {
		t.Errorf("Expected 3 configs, got %d", count)
	}

	// Очистка
	state.ClearConfigs(chatID)

	// Проверка после очистки
	count = state.GetConfigsCount(chatID)
	if count != 0 {
		t.Errorf("Expected 0 configs after clear, got %d", count)
	}

	_, exists := state.GetConfigs(chatID)
	if exists {
		t.Error("Configs should not exist after clear")
	}
}

func TestBotState_InvalidChatID(t *testing.T) {
	state := NewBotState()
	invalidChatID := int64(0)

	// Все операции с невалидным chatID должны возвращать безопасные значения
	if step := state.GetConfigStep(invalidChatID); step != StepNone {
		t.Errorf("GetConfigStep with invalid ID should return StepNone, got %v", step)
	}

	state.SetConfigStep(invalidChatID, StepWaitingServer)
	if step := state.GetConfigStep(invalidChatID); step != StepNone {
		t.Error("SetConfigStep should not work with invalid ID")
	}

	state.SetUserServer(invalidChatID, "example.com")
	_, exists := state.GetUserServer(invalidChatID)
	if exists {
		t.Error("GetUserServer should not return data for invalid ID")
	}

	state.SetUserName(invalidChatID, "testuser")
	_, exists = state.GetUserName(invalidChatID)
	if exists {
		t.Error("GetUserName should not return data for invalid ID")
	}

	config := ConfigData{Name: "test", Password: "test", Config: "test", Server: "test", Up: 0, Down: 0}
	if err := state.AddConfig(invalidChatID, config); err != ErrConfigLimitExceeded {
		t.Errorf("AddConfig with invalid ID should return ErrConfigLimitExceeded, got %v", err)
	}

	_, exists = state.GetConfigByIndex(invalidChatID, 0)
	if exists {
		t.Error("GetConfigByIndex should not return data for invalid ID")
	}

	state.ClearConfigs(invalidChatID)         // Не должно паниковать
	state.ClearUserConfigState(invalidChatID) // Не должно паниковать
}
