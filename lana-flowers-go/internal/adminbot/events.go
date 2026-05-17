package adminbot

// Типы событий маркетплейса, по которым админы получают нотификации и
// которые пишутся в event_log. Константы строковые потому что значение
// попадает в JSONB (admins.notifications) и в event_log.event_type —
// stringly-typed enum проще менять без миграций.

const (
	EventUserStarted    = "user_started"     // юзер открыл основной бот через /start
	EventUserRegistered = "user_registered"  // юзер прошёл онбординг полностью (есть phone)
	EventBouquetCreated = "bouquet_created"  // новое объявление в каталоге
	EventOfferCreated   = "offer_created"    // покупатель сделал предложение
	EventOfferAccepted  = "offer_accepted"   // продавец принял
	EventOfferRejected  = "offer_rejected"   // продавец отклонил
	EventOfferCountered = "offer_countered"  // встречная цена
	EventOfferCancelled = "offer_cancelled"  // отмена принятой сделки (любой стороной)
	EventOfferWithdrawn = "offer_withdrawn"  // покупатель отозвал свой pending
	EventOfferExpired   = "offer_expired"    // pending заэкспайрился по TTL
)

// AllEvents — порядок отображения в меню «Уведомления». Изменения здесь
// автоматически отражаются в UI меню (см. menu.go buildNotifMenu).
var AllEvents = []string{
	EventUserStarted,
	EventUserRegistered,
	EventBouquetCreated,
	EventOfferCreated,
	EventOfferAccepted,
	EventOfferRejected,
	EventOfferCountered,
	EventOfferCancelled,
	EventOfferWithdrawn,
	EventOfferExpired,
}

// eventLabels — человеческие подписи для UI. Короткие, помещаются в
// inline-кнопку Telegram (есть лимит ~64 байта на text кнопки).
var eventLabels = map[string]string{
	EventUserStarted:    "Новый юзер /start",
	EventUserRegistered: "Регистрация завершена",
	EventBouquetCreated: "Новое объявление",
	EventOfferCreated:   "Новое предложение",
	EventOfferAccepted:  "Сделка принята",
	EventOfferRejected:  "Сделка отклонена",
	EventOfferCountered: "Встречная цена",
	EventOfferCancelled: "Сделка отменена",
	EventOfferWithdrawn: "Предложение отозвано",
	EventOfferExpired:   "Предложение истекло",
}

// EventLabel — заголовок события для UI. Если тип неизвестен — возвращает
// сам строковый код (для устойчивости к рассинхрону между БД и Go-кодом).
func EventLabel(eventType string) string {
	if l, ok := eventLabels[eventType]; ok {
		return l
	}
	return eventType
}
