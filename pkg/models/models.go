package models

import (
	"errors"
	"hysconfigbot/pkg/consts"
	"sync"
)

// Ошибка при превышении лимита конфигов
var ErrConfigLimitExceeded = errors.New("достигнут лимит конфигов на пользователя")

// ConfigStep представляет текущий шаг создания конфига
type ConfigStep int

const (
	StepNone          ConfigStep = iota
	StepWaitingServer            // Ждём адрес сервера
	StepWaitingName              // Ждём имя пользователя
)

// ConfigData представляет данные конфига пользователя
type ConfigData struct {
	Name     string
	Password string
	Config   string
	Server   string
}

// UserConfigState хранит временные данные создания конфига
type UserConfigState struct {
	Server string
}

// BotState хранит состояние бота для каждого пользователя
type BotState struct {
	mu             sync.RWMutex
	WaitingForName map[int64]bool
	Configs        map[int64][]ConfigData     // Хранилище конфигов по chatID
	ConfigSteps    map[int64]ConfigStep       // Текущий шаг для каждого пользователя
	ConfigStates   map[int64]*UserConfigState // Временные данные создания конфига
}

// NewBotState создаёт новое состояние бота
func NewBotState() *BotState {
	return &BotState{
		WaitingForName: make(map[int64]bool),
		Configs:        make(map[int64][]ConfigData),
		ConfigSteps:    make(map[int64]ConfigStep),
		ConfigStates:   make(map[int64]*UserConfigState),
	}
}

// GetConfigStep возвращает текущий шаг для пользователя
func (s *BotState) GetConfigStep(chatID int64) ConfigStep {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ConfigSteps[chatID]
}

// SetConfigStep устанавливает текущий шаг для пользователя
func (s *BotState) SetConfigStep(chatID int64, step ConfigStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfigSteps[chatID] = step
}

// SetUserServer устанавливает адрес сервера для пользователя
func (s *BotState) SetUserServer(chatID int64, server string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConfigStates[chatID] == nil {
		s.ConfigStates[chatID] = &UserConfigState{}
	}
	s.ConfigStates[chatID].Server = server
}

// GetUserServer возвращает адрес сервера для пользователя
func (s *BotState) GetUserServer(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.ConfigStates[chatID]
	if !exists || state == nil {
		return "", false
	}
	return state.Server, true
}

// ClearUserConfigState очищает временные данные пользователя
func (s *BotState) ClearUserConfigState(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ConfigStates, chatID)
	delete(s.ConfigSteps, chatID)
}

// IsWaitingForName проверяет, ждёт ли бот имя от пользователя
func (s *BotState) IsWaitingForName(chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WaitingForName[chatID]
}

// SetWaitingForName устанавливает флаг ожидания имени
func (s *BotState) SetWaitingForName(chatID int64, waiting bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WaitingForName[chatID] = waiting
}

// AddConfig добавляет конфиг в хранилище
func (s *BotState) AddConfig(chatID int64, config ConfigData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Configs[chatID]) >= consts.MaxConfigsPerUser {
		return ErrConfigLimitExceeded
	}

	s.Configs[chatID] = append(s.Configs[chatID], config)
	return nil
}

// GetConfigs возвращает все конфиги пользователя
func (s *BotState) GetConfigs(chatID int64) ([]ConfigData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs, exists := s.Configs[chatID]
	return configs, exists
}

// GetConfigByIndex возвращает конфиг по индексу
func (s *BotState) GetConfigByIndex(chatID int64, index int) (ConfigData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs, exists := s.Configs[chatID]
	if !exists || index < 0 || index >= len(configs) {
		return ConfigData{}, false
	}
	return configs[index], true
}

// ClearConfigs удаляет все конфиги пользователя
func (s *BotState) ClearConfigs(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Configs, chatID)
}

// GetConfigsCount возвращает количество конфигов пользователя
func (s *BotState) GetConfigsCount(chatID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Configs[chatID])
}
