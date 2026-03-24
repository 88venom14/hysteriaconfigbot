# Hysteria2 Config Bot

Telegram-бот для генерации YAML-конфигурационных файлов Hysteria2/Mihomo.

[![Go Build](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Go Report](https://img.shields.io/badge/go%20report-A+-brightgreen)]()

## Структура проекта

```
hysconfigbot/
├── cmd/
│   └── bot/
│       └── main.go              # Точка входа + graceful shutdown
├── internal/
│   └── handlers/
│       └── handlers.go          # Обработчики команд и callback-ов бота
├── pkg/
│   ├── client/
│   │   └── client.go            # HTTP-клиент с поддержкой SOCKS5/HTTP прокси
│   ├── consts/
│   │   └── consts.go            # Константы, таймауты, сообщения бота
│   ├── generator/
│   │   ├── generator.go         # Генерация пароля и YAML-конфига
│   │   ├── generator_test.go    # Тесты генератора
│   │   └── sanitize_test.go     # Тесты санитизации имён файлов
│   └── models/
│       ├── models.go            # Модели данных, состояние бота, хранилище
│       └── models_test.go       # Тесты моделей и валидации
├── configs/
│   └── .env.example             # Пример конфигурации
├── .env                         # Конфигурация (не в git)
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Возможности

- ✅ Генерация YAML-конфигов для Hysteria2/Mihomo
- ✅ Уникальный пароль для каждого конфига
- ✅ Валидация имени пользователя (латиница + цифры)
- ✅ Валидация адреса сервера (домены, IP)
- ✅ Выбор скорости: авто или кастомный (UP/DOWN)
- ✅ Лимит конфигов: 10 на пользователя
- ✅ Хранение истории конфигов по chatID
- ✅ Скачивание конфигов в виде .yaml файлов
- ✅ Очистка истории конфигов
- ✅ Поддержка SOCKS5/HTTP прокси (для работы в РФ)
- ✅ Graceful shutdown (корректное завершение)
- ✅ Санитизация логов (скрытие токенов)
- ✅ Защита от race conditions (RWMutex)
- ✅ Inline-кнопки для удобной навигации

## Требования

- Go 1.26+
- Telegram Bot Token (получить у [@BotFather](https://t.me/BotFather))
- SOCKS5/HTTP прокси (опционально, для работы в РФ)

## Установка

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd hysconfigbot

# Установите зависимости
go mod tidy

# Скопируйте пример конфигурации
cp configs/.env.example .env

# Отредактируйте .env, добавив токен бота
```

## Настройка

### 1. Получение токена бота

1. Откройте [@BotFather](https://t.me/BotFather) в Telegram
2. Отправьте `/newbot`
3. Следуйте инструкциям
4. Скопируйте токен в `.env`:

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
```

### 2. Настройка прокси (если требуется)

Для России и других стран с блокировкой Telegram:

**NekoBox:**
```env
SOCKS5_PROXY=127.0.0.1:2080
```

**Clash Verge:**
```env
SOCKS5_PROXY=127.0.0.1:7890
```

**Tor:**
```env
SOCKS5_PROXY=127.0.0.1:9050
```

**HTTP/HTTPS прокси:**
```env
HTTPS_PROXY=http://proxy-host:port
```

## Запуск

```bash
# Разработка
go run cmd/bot/main.go

# Сборка
go build -o hysconfigbot.exe cmd/bot/main.go

# Запуск бинарника
./hysconfigbot.exe
```

## Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | 🚀 Запустить бота, показать приветственное сообщение |
| `/goconfig` | 🔑 Начать создание YAML-конфига Hysteria2 |
| `/config` | 📁 Показать список моих конфигов |
| `/help` | ❓ Показать справку по командам |
| `/stop` | ⏹ Отменить создание конфига |

## Процесс создания конфига

1. Отправьте `/goconfig`
2. Введите адрес сервера (например, `example.com`)
3. Введите имя пользователя (латиница, макс. 32 символа)
4. Выберите скорость:
   - **Авто** — без ограничений (0/0)
   - **Свой вариант** — введите в формате `UP DOWN` (например, `300 300`)
5. Получите:
   - Учётные данные (`username:password`)
   - YAML-конфиг в виде кода
   - Файл `.yaml` для скачивания

## Пример ответа

```
✅ Конфиг создан

alice: a3f5b8c9d2e1f4a7b6c8d9e0f1a2b3c4

```yaml
mixed-port: 7890
allow-lan: true
tcp-concurrent: true
...
proxies:
  - name: ⚡️ Hysteria2
    type: hysteria2
    server: example.com
    port: 443
    password: "alice:a3f5b8c9d2e1f4a7b6c8d9e0f1a2b3c4"
    up: 300
    down: 300

[Файл: alice.yaml]
```

## Функции

| ✅ Валидация имени пользователя | Только латиница и цифры (a-z, A-Z, 0-9) |
| ✅ Валидация адреса сервера | Домены, IP, без спецсимволов |
| ✅ Лимит конфигов | Максимум 10 на пользователя |
| ✅ Изоляция конфигов | Хранение по chatID |
| ✅ Санитизация имён файлов | Защита от path traversal |
| ✅ Защита от race conditions | RWMutex для потокобезопасности |
| ✅ Валидация chatID | Отсев некорректных идентификаторов |
| ✅ Graceful shutdown | Корректное завершение работы |
| ✅ Санитизация логов | Скрытие токенов и чувствительных данных |

## Тестирование

```bash
# Запустить все тесты
go test ./... -v

# Запустить с проверкой на race conditions
go test ./... -race

# Покрыть тестами конкретный пакет
go test ./pkg/generator/... -v
go test ./pkg/models/... -v
```

## Code Quality

```bash
# Сборка проекта
go build -o hysconfigbot.exe cmd/bot/main.go

# Проверка на ошибки
go vet ./...

# Форматирование кода
go fmt ./...

# Проверка на гонки данных
go build -race cmd/bot/main.go
```

## Структура конфига

Генерируемый YAML включает:

- **Proxy**: Hysteria2 с автоподключением
- **DNS**: Secure DNS (DoH) с Яндекс.DNS
- **TUN**: Виртуальный сетевой интерфейс
- **Sniffer**: Определение протоколов HTTP/TLS
- **Rule-providers**: Правила для обхода блокировок (Ru-Inline, Ru-Banned)
- **Proxy-groups**: Группа 🌍 VPN для выбора режима
