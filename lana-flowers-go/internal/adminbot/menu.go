package adminbot

import "fmt"

// Билдеры inline-клавиатур для меню админ-бота.
// Все callback_data имеют префиксы для роутинга в handleCallback:
//
//	"m:main"           — главное меню
//	"m:notif"          — меню «Уведомления»
//	"m:notif:toggle:<event_type>" — переключить toggle
//	"m:access"         — меню «Доступ» (заявки)
//	"m:access:grant:<tg_id>"  — выдать доступ конкретному
//	"m:access:deny:<tg_id>"   — отказать конкретному
//	"m:noop"           — заглушка для display-only кнопок

const (
	cbMain        = "m:main"
	cbNotif       = "m:notif"
	cbNotifToggle = "m:notif:toggle:" // + eventType
	cbAccess      = "m:access"
	cbAccessGrant = "m:access:grant:" // + tg_id
	cbAccessDeny  = "m:access:deny:"  // + tg_id
	cbStats       = "m:stats"
)

// mainMenu — корневое меню админа: три раздела.
func mainMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "📊 Статистика", CallbackData: cbStats}},
			{{Text: "🔔 Уведомления", CallbackData: cbNotif}},
			{{Text: "👤 Доступ к панели", CallbackData: cbAccess}},
		},
	}
}

// notifMenu — список toggle'ов по типам событий. Для каждого ключа из
// AllEvents строим строку с ✅/❌ в начале (отражает текущее состояние)
// и подписью события. Тап → переключение состояния.
func notifMenu(a *Admin) *InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(AllEvents)+1)
	for _, ev := range AllEvents {
		prefix := "❌"
		if a.EventEnabled(ev) {
			prefix = "✅"
		}
		rows = append(rows, []InlineKeyboardButton{{
			Text:         fmt.Sprintf("%s %s", prefix, EventLabel(ev)),
			CallbackData: cbNotifToggle + ev,
		}})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "← Назад", CallbackData: cbMain},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// accessMenu — список заявок на доступ. Каждая заявка — две кнопки в
// ряду: «Выдать» / «Отказать». Если заявок нет — пустой список + Назад.
func accessMenu(requests []Request) *InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(requests)*2+1)
	for _, r := range requests {
		label := r.DisplayName
		if r.Username != "" {
			label += " (@" + r.Username + ")"
		}
		if label == "" {
			label = "tg:" + r.TGID
		}
		// Один ряд — имя как display-only (callback заглушка), следующий ряд — действия.
		rows = append(rows, []InlineKeyboardButton{
			{Text: "👤 " + label, CallbackData: "m:noop"},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "✅ Выдать", CallbackData: cbAccessGrant + r.TGID},
			{Text: "❌ Отказать", CallbackData: cbAccessDeny + r.TGID},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "← Назад", CallbackData: cbMain},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

const (
	mainText   = "<b>Админ-панель Lana Flowers</b>\n\nВыберите раздел:"
	notifText  = "<b>Уведомления</b>\n\nТап по событию — переключить вкл/выкл.\n✅ — включено, ❌ — выключено."
	accessText = "<b>Доступ к панели</b>\n\nЗаявки на доступ (юзеры стартанули админ-бот):"
)

func accessEmptyText() string {
	return accessText + "\n\n<i>Заявок нет.</i>"
}

// statsMenu — single «Назад»-кнопка. Цифры в body сообщения, не на кнопках.
func statsMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "← Назад", CallbackData: cbMain}},
		},
	}
}

func statsText(s Stats) string {
	return fmt.Sprintf(
		"<b>📊 Статистика</b>\n\n"+
			"Нажали /start: <b>%d</b> (уникальных: <b>%d</b>)\n"+
			"Прошли регистрацию: <b>%d</b>\n"+
			"Опубликовано объявлений: <b>%d</b>\n"+
			"Делали предложения: <b>%d</b>",
		s.StartsTotal, s.StartsUnique, s.Registered, s.BouquetsPosted, s.OfferingBuyers,
	)
}
