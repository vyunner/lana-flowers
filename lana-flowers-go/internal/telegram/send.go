package telegram

import "log"

// ---- Формат сообщений ----
//
// Шаблон карточки в DM повторяет визуальную структуру in-app Сделок
// (lana-flowers-web/src/components/Deals.vue):
//
//   <emoji-статус> <Заголовок-действие>
//
//   <b>Название букета</b>
//
//   <b>NNN ₸</b> [· инлайн контекст вроде «−17% от вашей цены»]
//   [От: Имя]
//
//   [Инструкция/что делать]
//
//   [Inline-кнопки]
//
// Эмодзи в заголовке ОСТАВЛЕНЫ (в отличие от тостов и in-app): в чат-листе
// Telegram превью сообщения это единственный визуальный якорь чтобы
// различить «принято» / «отклонено» / «новое» не открывая бот. Без эмодзи
// все нотификации сливаются в одинаковый «Lana Flowers · текст…».
//
// А вот на callback-кнопках (Согласиться/Предложить/Отклонить) эмодзи
// убраны — текст совпадает с тем, что юзер видит в карточке Сделок.

// sendPhotoOrText — общая отправка с фоткой, фолбэк на текст. Если sendPhoto
// упал (Telegram не смог скачать картинку), отправляем без превью — лучше
// доставить нотификацию без фото, чем потерять её совсем.
func sendPhotoOrText(chatID int64, photoURL, text string, markup *InlineKeyboardMarkup) error {
	if photoURL != "" {
		_, err := SendPhoto(SendPhotoReq{
			ChatID:      chatID,
			Photo:       photoURL,
			Caption:     text,
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		if err == nil {
			return nil
		}
		log.Printf("notify: sendPhoto failed (%v), falling back to text", err)
	}
	_, err := SendMessage(SendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	return err
}

// sendStructured — HTML-сообщение с опциональной разметкой кнопок. Единый
// ParseMode и лог-формат для всех итоговых уведомлений без фото.
//
// ВАЖНО: typed nil pointer (*InlineKeyboardMarkup)(nil), приведённый к
// interface{}, превращается в JSON `null`, а Telegram на это отвечает
// «object expected as reply markup». Поэтому проверяем явно: если markup
// nil — вообще не ставим поле в req.
func sendStructured(chatID int64, text string, markup *InlineKeyboardMarkup, tag string) {
	req := SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}
	if markup != nil {
		req.ReplyMarkup = markup
	}
	if _, err := SendMessage(req); err != nil {
		log.Printf("notify %s chat=%d: %v", tag, chatID, err)
	}
}
