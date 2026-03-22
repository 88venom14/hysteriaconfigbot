package consts

import "time"

const (
	PollingTimeout = 60

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

const (
	BotWelcomeMessage    = "Привет! Я бот для генерации конфигов Hysteria2.\n\n📋 Доступные команды:\n🔑 /goconfig — создать конфиг\n📁 /config — мои конфиги\n❓ /help — справка"
	BotRequestServerMsg  = "🌐 Введите адрес сервера (например, Example.com):"
	BotRequestNameMsg    = "👤 Введите имя пользователя для генерации конфига:"
	BotConfigCreatedMsg  = "✅ Конфиг создан"
	BotErrorGenericMsg   = "❌ Произошла ошибка при генерации конфига. Попробуйте позже."
	BotWaitingForNameMsg = "Жду имя пользователя..."
	BotRetryButton       = "🔄 Сгенерировать заново"
	BotInvalidNameMsg    = "❌ Имя должно содержать только латинские буквы (a-z, A-Z) и цифры (0-9).\n\nПожалуйста, введите имя повторно:"
	BotInvalidServerMsg  = "❌ Неверный формат адреса сервера.\n\nАдрес должен быть доменным именем (например, example.com) или IP-адресом.\n\nПожалуйста, введите адрес повторно:"
	BotHelpMessage       = "📖 **Справка по командам**\n\n" +
		"🔹 /start — Запустить бота\n" +
		"🔑 /goconfig — Создать YAML-конфиг Hysteria2\n" +
		"📁 /config — Показать мои конфиги\n" +
		"❓ /help — Показать эту справку\n\n" +
		"💡 **Как использовать:**\n" +
		"1. Нажмите /goconfig\n" +
		"2. Введите адрес сервера\n" +
		"3. Введите имя пользователя (латиница)\n" +
		"4. Получите конфиг с уникальным паролем\n\n" +
		"📂 Все созданные конфиги доступны через /config"
	BotNoConfigsMessage    = "У вас пока нет созданных конфигов.\n\nИспользуйте /goconfig для создания."
	BotConfigsListMessage  = "📂 **Ваши конфиги:**\n\n"
	BotDownloadButton      = "📥 Скачать"
	BotClearConfirmMessage = "🗑 Вы уверены, что хотите удалить все ваши конфиги?\n\nБудет удалено конфигов: %d\n\nЭто действие нельзя отменить."
	BotClearDoneMessage    = "✅ Все конфиги удалены."
	BotClearButton         = "🗑 Да, удалить"
	BotCancelButton        = "❌ Отмена"
)
