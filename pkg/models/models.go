package models

import (
	"errors"
	"hysconfigbot/pkg/consts"
	"sync"
)

var ErrConfigLimitExceeded = errors.New("достигнут лимит конфигов на пользователя")

type ConfigStep int

const (
	StepNone ConfigStep = iota
	StepWaitingServer
	StepWaitingName
	StepWaitingSpeed
	StepWaitingCustomSpeed
)

type ConfigData struct {
	Name     string
	Password string
	Config   string
	Server   string
	Up       int
	Down     int
}

type UserConfigState struct {
	Server string
	Name   string
	Up     int
	Down   int
}

type BotState struct {
	mu             sync.RWMutex
	WaitingForName map[int64]bool
	Configs        map[int64][]ConfigData
	ConfigSteps    map[int64]ConfigStep
	ConfigStates   map[int64]*UserConfigState
}

func NewBotState() *BotState {
	return &BotState{
		WaitingForName: make(map[int64]bool),
		Configs:        make(map[int64][]ConfigData),
		ConfigSteps:    make(map[int64]ConfigStep),
		ConfigStates:   make(map[int64]*UserConfigState),
	}
}

func (s *BotState) GetConfigStep(chatID int64) ConfigStep {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ConfigSteps[chatID]
}

func (s *BotState) SetConfigStep(chatID int64, step ConfigStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfigSteps[chatID] = step
}

func (s *BotState) SetUserServer(chatID int64, server string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConfigStates[chatID] == nil {
		s.ConfigStates[chatID] = &UserConfigState{}
	}
	s.ConfigStates[chatID].Server = server
}

func (s *BotState) SetUserName(chatID int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConfigStates[chatID] == nil {
		s.ConfigStates[chatID] = &UserConfigState{}
	}
	s.ConfigStates[chatID].Name = name
}

func (s *BotState) SetUserSpeed(chatID int64, up, down int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConfigStates[chatID] == nil {
		s.ConfigStates[chatID] = &UserConfigState{}
	}
	s.ConfigStates[chatID].Up = up
	s.ConfigStates[chatID].Down = down
}

func (s *BotState) GetUserServer(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.ConfigStates[chatID]
	if !exists || state == nil {
		return "", false
	}
	return state.Server, true
}

func (s *BotState) GetUserName(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.ConfigStates[chatID]
	if !exists || state == nil {
		return "", false
	}
	return state.Name, true
}

func (s *BotState) GetUserSpeed(chatID int64) (int, int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.ConfigStates[chatID]
	if !exists || state == nil {
		return 0, 0, false
	}
	return state.Up, state.Down, true
}

func (s *BotState) ClearUserConfigState(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ConfigStates, chatID)
	delete(s.ConfigSteps, chatID)
}

func (s *BotState) IsWaitingForName(chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WaitingForName[chatID]
}

func (s *BotState) SetWaitingForName(chatID int64, waiting bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WaitingForName[chatID] = waiting
}

func (s *BotState) AddConfig(chatID int64, config ConfigData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Configs[chatID]) >= consts.MaxConfigsPerUser {
		return ErrConfigLimitExceeded
	}

	s.Configs[chatID] = append(s.Configs[chatID], config)
	return nil
}

func (s *BotState) GetConfigs(chatID int64) ([]ConfigData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs, exists := s.Configs[chatID]
	return configs, exists
}

func (s *BotState) GetConfigByIndex(chatID int64, index int) (ConfigData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs, exists := s.Configs[chatID]
	if !exists || index < 0 || index >= len(configs) {
		return ConfigData{}, false
	}
	return configs[index], true
}

func (s *BotState) ClearConfigs(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Configs, chatID)
}

func (s *BotState) GetConfigsCount(chatID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Configs[chatID])
}
