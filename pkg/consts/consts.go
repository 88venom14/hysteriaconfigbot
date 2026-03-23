package consts

import "time"

const (
	PollingTimeout = 60 * time.Second

	HTTPTimeout         = 120 * time.Second
	TLSHandshakeTimeout = 60 * time.Second
	ResponseTimeout     = 60 * time.Second
)

const (
	PasswordByteLength = 16

	DefaultPort = 443

	MaxNameLength = 32

	MaxServerLength = 253
)

const (
	MaxConfigsPerUser = 10
)

// DNS серверы по умолчанию
var DefaultDNSServers = []string{"tls://77.88.8.8", "195.208.4.1"}

const (
	BotWelcomeMessage        = "Привет! Я бот для генерации конфигов Hysteria2.\n\n📋 Доступные команды:\n🔑 /goconfig — создать конфиг\n📁 /config — мои конфиги\n❓ /help — справка"
	BotRequestServerMsg      = "🌐 Введите адрес сервера (например, Example.com):"
	BotRequestNameMsg        = "👤 Введите имя пользователя для генерации конфига:"
	BotRequestSpeedMsg       = "📶 Выберите скорость интернета:\n\nВыберите предложенные варианты"
	BotConfigCreatedMsg      = "✅ Конфиг создан"
	BotErrorGenericMsg       = "❌ Произошла ошибка при генерации конфига. Попробуйте позже."
	BotRetryButton           = "🔄 Сгенерировать заново"
	BotInvalidNameMsg        = "❌ Имя должно содержать только латинские буквы (a-z, A-Z) и цифры (0-9).\n\nПожалуйста, введите имя повторно:"
	BotInvalidServerMsg      = "❌ Неверный формат адреса сервера.\n\nАдрес должен быть доменным именем (например, example.com) или IP-адресом.\n\nПожалуйста, введите адрес повторно:"
	BotInvalidSpeedMsg       = "❌ Неверный номер скорости. Выберите вариант 1 или 2:\n\n1. 🤖 Авто (не ограничивать)\n2. 🔧 Свой вариант"
	BotRequestCustomSpeedMsg = "📶 Введите вашу скорость в формате: UP DOWN (например, 300 300):\n\nUP — исходящая скорость (Мбит/с)\nDOWN — входящая скорость (Мбит/с)"
	BotInvalidCustomSpeedMsg = "❌ Неверный формат. Введите два числа через пробел (например, 300 300):"
	BotStopMessage           = "❌ Создание конфига отменено.\n\nИспользуйте /goconfig для начала создания нового конфига."
	BotHelpMessage           = "📖 **Справка по командам**\n\n" +
		"🔹 /start — Запустить бота\n" +
		"🔑 /goconfig — Создать YAML-конфиг Hysteria2\n" +
		"📁 /config — Показать мои конфиги\n" +
		"❓ /help — Показать эту справку\n" +
		"⏹ /stop — Отменить создание конфига\n\n" +
		"💡 **Как использовать:**\n" +
		"1. Нажмите /goconfig\n" +
		"2. Введите адрес сервера\n" +
		"3. Введите имя пользователя (латиница)\n" +
		"4. Выберите скорость интернета\n" +
		"5. Получите конфиг с уникальным паролем\n\n" +
		"📂 Все созданные конфиги доступны через /config"
	BotNoConfigsMessage    = "У вас пока нет созданных конфигов.\n\nИспользуйте /goconfig для создания."
	BotConfigsListMessage  = "📂 **Ваши конфиги:**\n\n"
	BotDownloadButton      = "📥 Скачать"
	BotClearConfirmMessage = "🗑 Вы уверены, что хотите удалить все ваши конфиги?\n\nБудет удалено конфигов: %d\n\nЭто действие нельзя отменить."
	BotClearDoneMessage    = "✅ Все конфиги удалены."
	BotClearButton         = "🗑 Да, удалить"
	BotCancelButton        = "❌ Отмена"
)
