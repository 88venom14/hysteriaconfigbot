# Отчёты по Code Review: Hysteria2 Config Bot

**Дата проведения:** 22 марта 2026 г.  
**Статус:** 🔴 Требуются исправления перед production

---

## 📊 Сводка

| Категория | Количество |
|-----------|------------|
| 🔴 Critical | 4 |
| 🟡 Suggestion | 6 |
| 🟢 Nice to have | 4 |
| **Всего** | **14** |

---

## 🔴 Critical Issues (Обязательно к исправлению)

### 1. Уязвимость Path Traversal в загрузке конфигов

**Файл:** `internal/handlers/handlers.go`  
**Строки:** 337-348  
**Серьёзность:** 🔴 **CRITICAL** — Возможность записи файлов за пределы разрешённой директории

#### Описание проблемы

Функция `HandleDownload` использует имя пользователя (`cfg.Name`) напрямую для создания имени файла:

```go
func (h *Handler) HandleDownload(chatID int64, configIndex int) {
    cfg, exists := h.State.GetConfigByIndex(chatID, configIndex)
    if !exists {
        // ...
    }

    if err := h.sendConfigFile(chatID, cfg.Name, cfg.Config); err != nil {
        // ...
    }
}
```

Функция `sendConfigFile` создаёт файл в temp-директории:

```go
func (h *Handler) sendConfigFile(chatID int64, userName, config string) error {
    fileName := fmt.Sprintf("%s_config.yaml", userName)  // ← userName используется напрямую
    tmpDir := os.TempDir()
    tmpFilePath := filepath.Join(tmpDir, fileName)
    // ...
}
```

**Атака:** Если злоумышленник сможет создать конфиг с именем `../../../etc/passwd`, файл будет записан вне temp-директории.

Хотя `IsValidLatinName()` ограничивает символы, полагаться только на это недостаточно — нужна дополнительная санитизация на уровне файловой системы.

#### Как исправить

**Шаг 1:** Создать функцию санитизации имени файла

Создайте новый файл `pkg/generator/sanitize.go` или добавьте в `pkg/generator/generator.go`:

```go
// SanitizeFileName удаляет опасные символы из имени файла
// Возвращает безопасное имя для использования в файловой системе
func SanitizeFileName(name string) string {
    // Заменяем потенциально опасные символы
    sanitized := strings.ReplaceAll(name, "/", "")
    sanitized = strings.ReplaceAll(sanitized, "\\", "")
    sanitized = strings.ReplaceAll(sanitized, "..", "")
    sanitized = strings.ReplaceAll(sanitized, "\x00", "") // null byte
    
    // Оставляем только безопасные символы: буквы, цифры, дефис, подчёркивание
    var result strings.Builder
    for _, r := range sanitized {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
           (r >= '0' && r <= '9') || r == '-' || r == '_' {
            result.WriteRune(r)
        } else {
            result.WriteRune('_') // Заменяем небезопасные символы на _
        }
    }
    
    // Гарантируем непустое имя
    if result.Len() == 0 {
        return "config"
    }
    
    return result.String()
}
```

**Шаг 2:** Применить санитизацию в `sendConfigFile`

В файле `internal/handlers/handlers.go` измените функцию:

```go
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
```

**Шаг 3:** Добавить тесты

Создайте файл `pkg/generator/sanitize_test.go`:

```go
package generator

import "testing"

func TestSanitizeFileName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"normal", "alice", "alice"},
        {"with_spaces", "alice bob", "alice_bob"},
        {"path_traversal", "../../../etc/passwd", "etcpasswd"},
        {"backslash", "user\\name", "username"},
        {"null_byte", "user\x00name", "user_name"},
        {"special_chars", "user@name#", "user_name"},
        {"empty", "", "config"},
        {"only_special", "@#$", "config"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := SanitizeFileName(tt.input)
            if result != tt.expected {
                t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

**Проверка:**
```bash
go test ./pkg/generator/... -v
```

---

### 2. Race Condition при доступе к конфигу

**Файл:** `internal/handlers/handlers.go`, `pkg/models/models.go`  
**Строки:** 337-348 (handlers), 157-162 (models)  
**Серьёзность:** 🔴 **CRITICAL** — Возможна отправка несуществующего конфига или утечка памяти

#### Описание проблемы

```go
// В HandleDownload:
cfg, exists := h.State.GetConfigByIndex(chatID, configIndex)  // ← Чтение с RLock
if !exists {
    // ...
}

// RLock уже освобождён здесь!
if err := h.sendConfigFile(chatID, cfg.Name, cfg.Config); err != nil {  // ← Использование данных
    // ...
}
```

**Сценарий гонки:**
1. Пользователь запрашивает конфиг #3
2. `GetConfigByIndex` возвращает конфиг
3. **До отправки файла** пользователь отправляет `/clear` (другой горутине)
4. Конфиги удаляются из памяти
5. `sendConfigFile` работает с уже невалидными данными

#### Как исправить

**Вариант 1: Копирование данных перед освобождением блокировки** (рекомендуется)

В файле `pkg/models/models.go` добавьте метод для безопасного копирования конфига:

```go
// GetConfigByIndexCopy возвращает копию конфига для безопасного использования вне блокировки
func (s *BotState) GetConfigByIndexCopy(chatID int64, index int) (ConfigData, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    configs, exists := s.Configs[chatID]
    if !exists || index < 0 || index >= len(configs) {
        return ConfigData{}, false
    }
    
    // Возвращаем копию структуры (ConfigData — value type, копируется автоматически)
    return configs[index], true
}
```

В файле `internal/handlers/handlers.go` обновите `HandleDownload`:

```go
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
```

**Вариант 2: Использование контекста для отмены** (более сложная реализация)

Если требуется гарантировать, что конфиг не будет удалён во время отправки:

```go
// В models.go добавьте
type ConfigLock struct {
    mu sync.RWMutex
    inUse map[int64]map[int]bool  // chatID -> set of config indices in use
}

// Методы ReserveConfig/ReleaseConfig для блокировки конфига на время использования
```

**Рекомендация:** Используйте Вариант 1 — он проще и достаточно надёжен для данного случая.

**Проверка:**
```bash
# Запуск с детектором гонок
go run -race cmd/bot/main.go

# Тестирование в двух сессиях:
# 1. Запросить скачивание конфига
# 2. Сразу отправить /clear
# 3. Убедиться, что нет паники и файл корректно отправляется
```

---

### 3. Отсутствие персистентности данных

**Файл:** `pkg/models/models.go`  
**Строки:** 57-63  
**Серьёзность:** 🔴 **CRITICAL** — Потеря всех данных при перезапуске бота

#### Описание проблемы

```go
func NewBotState() *BotState {
    return &BotState{
        WaitingForName: make(map[int64]bool),
        Configs:        make(map[int64][]ConfigData),  // ← Только в памяти
        ConfigSteps:    make(map[int64]ConfigStep),
        ConfigStates:   make(map[int64]*UserConfigState),
    }
}
```

**Последствия:**
- При перезапуске бота все пользователи теряют свои конфиги
- При деплое на сервер все данные теряются
- Невозможно масштабировать на несколько инстансов

#### Как исправить

**Вариант 1: SQLite (рекомендуется для простоты)**

**Шаг 1:** Добавить зависимость

```bash
go get github.com/mattn/go-sqlite3
```

**Шаг 2:** Создать пакет для работы с БД

Создайте файл `pkg/storage/sqlite.go`:

```go
package storage

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "sync"

    _ "github.com/mattn/go-sqlite3"
    "hysconfigbot/pkg/models"
)

type SQLiteStorage struct {
    db *sql.DB
    mu sync.RWMutex
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Создаём таблицы
    schema := `
    CREATE TABLE IF NOT EXISTS user_configs (
        chat_id INTEGER NOT NULL,
        config_index INTEGER NOT NULL,
        name TEXT NOT NULL,
        password TEXT NOT NULL,
        config TEXT NOT NULL,
        server TEXT NOT NULL,
        up INTEGER NOT NULL,
        down INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (chat_id, config_index)
    );

    CREATE TABLE IF NOT EXISTS user_states (
        chat_id INTEGER PRIMARY KEY,
        server TEXT,
        name TEXT,
        up INTEGER,
        down INTEGER,
        step INTEGER DEFAULT 0
    );

    CREATE INDEX IF NOT EXISTS idx_chat_id ON user_configs(chat_id);
    `

    if _, err := db.Exec(schema); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to create schema: %w", err)
    }

    return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Close() error {
    return s.db.Close()
}

// Сохранение конфига
func (s *SQLiteStorage) AddConfig(chatID int64, config models.ConfigData) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Получаем текущий индекс
    var maxIndex int
    err := s.db.QueryRow(`
        SELECT COALESCE(MAX(config_index), -1) FROM user_configs WHERE chat_id = ?
    `, chatID).Scan(&maxIndex)
    
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf("failed to get max index: %w", err)
    }

    // Проверка лимита
    var count int
    err = s.db.QueryRow(`
        SELECT COUNT(*) FROM user_configs WHERE chat_id = ?
    `, chatID).Scan(&count)
    
    if err != nil {
        return fmt.Errorf("failed to count configs: %w", err)
    }
    
    if count >= 10 {
        return models.ErrConfigLimitExceeded
    }

    // Вставляем новый конфиг
    _, err = s.db.Exec(`
        INSERT INTO user_configs (chat_id, config_index, name, password, config, server, up, down)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, chatID, maxIndex+1, config.Name, config.Password, config.Config, config.Server, config.Up, config.Down)

    if err != nil {
        return fmt.Errorf("failed to insert config: %w", err)
    }

    return nil
}

// Получение всех конфигов пользователя
func (s *SQLiteStorage) GetConfigs(chatID int64) ([]models.ConfigData, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    rows, err := s.db.Query(`
        SELECT name, password, config, server, up, down 
        FROM user_configs 
        WHERE chat_id = ? 
        ORDER BY config_index
    `, chatID)
    
    if err != nil {
        return nil, fmt.Errorf("failed to query configs: %w", err)
    }
    defer rows.Close()

    var configs []models.ConfigData
    for rows.Next() {
        var cfg models.ConfigData
        if err := rows.Scan(&cfg.Name, &cfg.Password, &cfg.Config, &cfg.Server, &cfg.Up, &cfg.Down); err != nil {
            return nil, fmt.Errorf("failed to scan config: %w", err)
        }
        configs = append(configs, cfg)
    }

    return configs, rows.Err()
}

// Удаление всех конфигов пользователя
func (s *SQLiteStorage) ClearConfigs(chatID int64) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    _, err := s.db.Exec(`DELETE FROM user_configs WHERE chat_id = ?`, chatID)
    return err
}

// Сохранение состояния (step)
func (s *SQLiteStorage) SetConfigStep(chatID int64, step models.ConfigStep) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO user_states (chat_id, step) VALUES (?, ?)
    `, chatID, step)
    
    return err
}

// Получение состояния
func (s *SQLiteStorage) GetConfigStep(chatID int64) (models.ConfigStep, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var step int
    err := s.db.QueryRow(`SELECT step FROM user_states WHERE chat_id = ?`, chatID).Scan(&step)
    
    if err == sql.ErrNoRows {
        return models.StepNone, nil
    }
    if err != nil {
        return models.StepNone, fmt.Errorf("failed to get step: %w", err)
    }

    return models.ConfigStep(step), nil
}

// Сохранение временных данных (server, name, speed)
func (s *SQLiteStorage) SetUserServer(chatID int64, server string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO user_states (chat_id, server) VALUES (?, ?)
    `, chatID, server)
    
    return err
}

func (s *SQLiteStorage) GetUserServer(chatID int64) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var server sql.NullString
    err := s.db.QueryRow(`SELECT server FROM user_states WHERE chat_id = ?`, chatID).Scan(&server)
    
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", fmt.Errorf("failed to get server: %w", err)
    }

    if !server.Valid {
        return "", nil
    }
    return server.String, nil
}

// Аналогично для name, up, down...
```

**Шаг 3:** Обновить `BotState` для использования хранилища

В файле `pkg/models/models.go` добавьте интерфейс:

```go
// Storage определяет интерфейс для хранилища данных
type Storage interface {
    AddConfig(chatID int64, config ConfigData) error
    GetConfigs(chatID int64) ([]ConfigData, error)
    ClearConfigs(chatID int64) error
    SetConfigStep(chatID int64, step ConfigStep) error
    GetConfigStep(chatID int64) (ConfigStep, error)
    SetUserServer(chatID int64, server string) error
    GetUserServer(chatID int64) (string, error)
    // ... другие методы
    Close() error
}
```

Обновите `BotState`:

```go
type BotState struct {
    storage Storage  // ← Добавляем хранилище
    
    // Кэш в памяти для быстрых операций (опционально)
    mu           sync.RWMutex
    ConfigSteps  map[int64]ConfigStep
    ConfigStates map[int64]*UserConfigState
}

func NewBotState(storage Storage) *BotState {
    return &BotState{
        storage:      storage,
        ConfigSteps:  make(map[int64]ConfigStep),
        ConfigStates: make(map[int64]*UserConfigState),
    }
}
```

**Шаг 4:** Обновить `main.go`

В файле `cmd/bot/main.go`:

```go
func main() {
    loadEnv()

    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        log.Fatalf("[FATAL] Bot initialization failed: missing token")
    }

    // Инициализация хранилища
    dbPath := os.Getenv("DB_PATH")
    if dbPath == "" {
        dbPath = "bot_data.db"
    }
    
    storage, err := storage.NewSQLiteStorage(dbPath)
    if err != nil {
        log.Fatalf("[FATAL] Failed to initialize storage: %v", err)
    }
    defer storage.Close()

    httpClient := client.NewHTTPClient()

    bot, err := tgbotapi.NewBotAPIWithClient(token, "https://api.telegram.org/bot%s/%s", httpClient)
    if err != nil {
        log.Fatalf("[FATAL] Failed to connect to Telegram API: %v", err)
    }

    // ... настройка команд ...

    log.Printf("[INFO] Bot started: %s", bot.Self.UserName)

    // Передаём storage в BotState
    botState := models.NewBotState(storage)
    handler := handlers.NewHandler(bot, botState)

    // ... основной цикл ...
}
```

**Вариант 2: JSON-файл (проще, но менее надёжно)**

Если SQLite слишком сложен, можно использовать JSON-файл:

```go
// pkg/storage/json.go
func (s *JSONStorage) Save() error {
    data, err := json.MarshalIndent(s.data, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(s.path, data, 0644)
}
```

**Проверка:**
```bash
# После изменений
go build -o hysconfigbot.exe cmd/bot/main.go

# Запустить бота, создать конфиги
# Перезапустить бота
# Убедиться, что конфиги сохранились через /config
```

---

### 4. Отсутствие валидации chatID

**Файл:** `pkg/models/models.go`  
**Серьёзность:** 🔴 **HIGH** — Возможны некорректные ключи в мапах

#### Описание проблемы

```go
type BotState struct {
    Configs map[int64][]ConfigData  // ← chatID используется как ключ без валидации
}
```

**Проблемы:**
- Отрицательные chatID возможны (каналы имеют формат `-100xxxxxxxxxx`)
- Нет проверки на `0` (невалидный chatID)
- Нет верхнего предела

#### Как исправить

Добавить функцию валидации в `pkg/models/models.go`:

```go
// IsValidChatID проверяет корректность chatID
// Telegram chatID может быть:
// - Положительным (личные сообщения)
// - Отрицательным (группы, супергруппы)
// - Формата -100xxxxxxxxxx (каналы, супергруппы)
func IsValidChatID(chatID int64) bool {
    // Исключаем 0 и слишком маленькие значения
    if chatID == 0 {
        return false
    }
    
    // Абсолютное значение должно быть разумным (Telegram использует до 2^63-1)
    absID := chatID
    if absID < 0 {
        absID = -absID
    }
    
    // Минимальный разумный chatID в Telegram
    if absID < 10000 {
        return false
    }
    
    return true
}
```

Использовать во всех методах, работающих с chatID:

```go
func (s *BotState) GetConfigStep(chatID int64) ConfigStep {
    if !IsValidChatID(chatID) {
        log.Printf("[WARN] Invalid chatID: %d", chatID)
        return StepNone
    }
    
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.ConfigSteps[chatID]
}
```

**Проверка:**
```bash
# Добавить unit-тест
go test ./pkg/models/... -v -run TestIsValidChatID
```

---

## 🟡 Suggestion Issues (Рекомендуется к исправлению)

### 5. Дублирование кода обработки ошибок

**Файл:** `internal/handlers/handlers.go`  
**Строки:** 87-92, 103-108, 115-120

#### Описание проблемы

Одинаковый код повторяется многократно:

```go
if !generator.IsValidServerAddress(server) {
    msg := tgbotapi.NewMessage(chatID, consts.BotInvalidServerMsg)
    msg.ParseMode = tgbotapi.ModeMarkdown
    h.send(msg)
    return
}
```

#### Как исправить

Создать helper-функции в `internal/handlers/handlers.go`:

```go
// sendError отправляет сообщение об ошибке пользователю
func (h *Handler) sendError(chatID int64, message string) {
    msg := tgbotapi.NewMessage(chatID, message)
    msg.ParseMode = tgbotapi.ModeMarkdown
    h.send(msg)
}

// sendAndLogError отправляет ошибку и логирует её
func (h *Handler) sendAndLogError(chatID int64, logMsg string, userMsg string) {
    log.Printf(logMsg)
    h.sendError(chatID, userMsg)
}
```

Заменить все повторения:

```go
// Было:
if !generator.IsValidServerAddress(server) {
    msg := tgbotapi.NewMessage(chatID, consts.BotInvalidServerMsg)
    msg.ParseMode = tgbotapi.ModeMarkdown
    h.send(msg)
    return
}

// Стало:
if !generator.IsValidServerAddress(server) {
    h.sendError(chatID, consts.BotInvalidServerMsg)
    return
}
```

**Проверка:**
```bash
go build ./...
# Убедиться, что нет ошибок компиляции
```

---

### 6. Неиспользуемое поле WaitingForName

**Файл:** `pkg/models/models.go`  
**Строки:** 50-55

#### Описание проблемы

```go
type BotState struct {
    mu             sync.RWMutex
    WaitingForName map[int64]bool  // ← Объявлено, но не используется
    Configs        map[int64][]ConfigData
    ConfigSteps    map[int64]ConfigStep
    ConfigStates   map[int64]*UserConfigState
}
```

Методы существуют, но не вызываются:
- `IsWaitingForName()` — 0 вызовов
- `SetWaitingForName()` — 0 вызовов

#### Как исправить

Удалить мёртвый код:

```go
type BotState struct {
    mu           sync.RWMutex
    Configs      map[int64][]ConfigData
    ConfigSteps  map[int64]ConfigStep
    ConfigStates map[int64]*UserConfigState
}

func NewBotState() *BotState {
    return &BotState{
        Configs:      make(map[int64][]ConfigData),
        ConfigSteps:  make(map[int64]ConfigStep),
        ConfigStates: make(map[int64]*UserConfigState),
    }
}
```

Удалить методы:
- `IsWaitingForName()`
- `SetWaitingForName()`

**Проверка:**
```bash
grep -r "WaitingForName" .
# Должен вернуть только это определение
go build ./...
```

---

### 7. Магические числа в шаблоне конфигурации

**Файл:** `pkg/generator/generator.go`  
**Строки:** 47-50

#### Описание проблемы

```yaml
dns:
  default-nameserver: [tls://77.88.8.8, 195.208.4.1]
  proxy-server-nameserver: [tls://77.88.8.8, 195.208.4.1]
  direct-nameserver: [tls://77.88.8.8, 195.208.4.1]
```

DNS-серверы захардкожены. Для пользователей из других стран могут быть недоступны.

#### Как исправить

**Вариант 1:** Вынести в константы

В `pkg/consts/consts.go`:

```go
var (
    DefaultDNSServers = []string{"tls://77.88.8.8", "195.208.4.1"}
)
```

В `pkg/generator/generator.go`:

```go
type ConfigParams struct {
    Server      string
    Port        int
    Password    string
    Up          int
    Down        int
    DNSServers  []string  // ← Добавляем
}

// В GenerateConfig:
if params.DNSServers == nil {
    params.DNSServers = consts.DefaultDNSServers
}
```

**Вариант 2:** Читать из .env

В `.env.example`:

```env
# DNS серверы (опционально)
DNS_SERVERS=tls://77.88.8.8,195.208.4.1
```

В `main.go`:

```go
dnsServers := strings.Split(os.Getenv("DNS_SERVERS"), ",")
```

**Проверка:**
```bash
# Создать конфиг, проверить что DNS подставляются корректно
```

---

### 8. Неполная обработка ошибок в loadEnv

**Файл:** `cmd/bot/main.go`  
**Строки:** 19-40

#### Описание проблемы

```go
func loadEnv() {
    // ... попытки загрузить .env ...
    log.Printf("[WARN] .env file not found, using environment variables")
}

func main() {
    loadEnv()
    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        log.Fatalf("[FATAL] Bot initialization failed: missing token")
    }
}
```

Если `.env` не найден, бот продолжает работу, но затем крашится из-за отсутствия токена.

#### Как исправить

```go
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
    
    // ...
}
```

---

### 9. Отсутствие timeout на отправку сообщений

**Файл:** `internal/handlers/handlers.go`  
**Строки:** 309-315

#### Описание проблемы

```go
func (h *Handler) send(msg tgbotapi.Chattable) {
    result, err := h.Bot.Send(msg)  // ← Может заблокироваться навсегда
    if err != nil {
        log.Printf("[ERROR] Failed to send message: %v", err)
    }
}
```

#### Как исправить

Использовать контекст с timeout:

```go
func (h *Handler) send(msg tgbotapi.Chattable) {
    // Создаём канал для результата
    done := make(chan error, 1)
    
    go func() {
        _, err := h.Bot.Send(msg)
        done <- err
    }()
    
    // Ждём максимум 30 секунд
    select {
    case err := <-done:
        if err != nil {
            log.Printf("[ERROR] Failed to send message: %v", err)
        }
    case <-time.After(30 * time.Second):
        log.Printf("[ERROR] Message send timeout")
    }
}
```

Или настроить timeout в HTTP-клиенте (уже сделано в `client.go`).

---

### 10. Константа без единицы измерения

**Файл:** `pkg/consts/consts.go`  
**Строка:** 6

#### Описание проблемы

```go
const (
    PollingTimeout = 60  // ← Неясно: секунды? минуты?
)
```

#### Как исправить

```go
const (
    PollingTimeout = 60 * time.Second
)
```

В `main.go`:

```go
updateConfig.Timeout = consts.PollingTimeout.Seconds()  // Если нужно int
// или
updateConfig.Timeout = consts.PollingTimeout  // Если поле типа time.Duration
```

---

## 🟢 Nice to have (Опционально)

### 11. Отсутствие graceful shutdown

**Файл:** `cmd/bot/main.go`

#### Рекомендация

Добавить обработку сигналов:

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // ... инициализация ...
    
    // Канал для сигналов
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Запуск бота в горутине
    go func() {
        for update := range updates {
            // ... обработка ...
        }
    }()
    
    // Ждём сигнал завершения
    <-sigChan
    log.Printf("[INFO] Shutting down...")
    
    // Закрываем хранилище
    if storage != nil {
        storage.Close()
    }
}
```

---

### 12. Логирование чувствительных данных

**Файл:** `cmd/bot/main.go`  
**Строка:** 94

#### Описание проблемы

```go
log.Printf("[INFO] [CHAT_ID:%d] Received message: %s", chatID, text)
```

Если пользователь введёт токен или пароль, это будет залогировано.

#### Рекомендация

```go
func sanitizeForLog(text string) string {
    if len(text) > 50 {
        return text[:50] + "..."
    }
    // Маскируем потенциальные токены (длинные строки)
    if len(text) > 30 && strings.Contains(text, ":") {
        return "[REDACTED]"
    }
    return text
}

// В main.go:
log.Printf("[INFO] [CHAT_ID:%d] Received message: %s", chatID, sanitizeForLog(text))
```

---

### 13. Отсутствие unit-тестов

#### Рекомендация

Создать тесты:

```
pkg/generator/
├── generator.go
├── generator_test.go    # ← Добавить
└── sanitize_test.go     # ← Добавить

pkg/models/
├── models.go
└── models_test.go       # ← Добавить
```

**Пример теста для GeneratePassword:**

```go
func TestGeneratePassword(t *testing.T) {
    // Проверка длины
    password, err := GeneratePassword()
    if err != nil {
        t.Fatalf("GeneratePassword() error: %v", err)
    }
    
    expectedLen := consts.PasswordByteLength * 2  // hex encoding
    if len(password) != expectedLen {
        t.Errorf("Password length = %d, want %d", len(password), expectedLen)
    }
    
    // Проверка уникальности
    passwords := make(map[string]bool)
    for i := 0; i < 1000; i++ {
        p, _ := GeneratePassword()
        if passwords[p] {
            t.Error("Duplicate password generated")
        }
        passwords[p] = true
    }
}
```

**Запуск тестов:**
```bash
go test ./... -v
go test ./... -race  # Проверка на гонки
```

---

### 14. Неиспользуемые импорты

**Файл:** `pkg/generator/generator.go`  
**Строка:** 5

#### Описание проблемы

```go
import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "hysconfigbot/pkg/consts"
    "io"              // ← Используется только для rand.Reader
    "strings"
    "text/template"
)
```

#### Рекомендация

Можно упростить:

```go
func GeneratePassword() (string, error) {
    bytes := make([]byte, consts.PasswordByteLength)
    if _, err := rand.Read(bytes); err != nil {  // ←直接使用 crypto/rand
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
```

---

## 📋 Чек-лист исправлений

### Priority 1 (Critical)
- [ ] 1. Path Traversal — добавить `SanitizeFileName()`
- [ ] 2. Race Condition — добавить `GetConfigByIndexCopy()`
- [ ] 3. Персистентность — добавить SQLite/JSON хранилище
- [ ] 4. Валидация chatID — добавить `IsValidChatID()`

### Priority 2 (Suggestion)
- [ ] 5. Дублирование кода — создать helper-функции
- [ ] 6. Удалить `WaitingForName`
- [ ] 7. DNS серверы — вынести в конфиг
- [ ] 8. Улучшить обработку ошибок `loadEnv`
- [ ] 9. Timeout на отправку сообщений
- [ ] 10. Единицы измерения в константах

### Priority 3 (Nice to have)
- [ ] 11. Graceful shutdown
- [ ] 12. Санитизация логов
- [ ] 13. Unit-тесты
- [ ] 14. Упростить импорты

---

## 🔧 Команды для проверки

```bash
# Сборка
go build -o hysconfigbot.exe cmd/bot/main.go

# Тесты
go test ./... -v

# Проверка на гонки
go run -race cmd/bot/main.go

# Линтер
go vet ./...

# Форматирование
go fmt ./...
```

---

## 📚 Дополнительные ресурсы

- [Go Security Best Practices](https://github.com/Checkmarx/Go-SEC)
- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [SQLite for Go](https://github.com/mattn/go-sqlite3)
