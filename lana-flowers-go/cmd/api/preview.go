package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"lana-flowers-go/internal/telegram"
)

// runPreviewCmd шлёт все 8 типов нотификаций с тестовыми данными на
// указанный chat_id. Цель — глазами проверить визуальный рендеринг
// (line-wrap, NBSP, иерархия label:value, CTA-кнопки) без необходимости
// гонять реальный торг через мини-апп.
//
// Usage: ./app preview <tg_id>
//
// callback_data в превью-кнопках имеют формат "preview:noop" — обработчик
// webhook'а их не знает, тапы по ним ничего не сделают (только Telegram
// покажет "загрузка"). Это намеренно — превью не должно создавать оффер.
func runPreviewCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: app preview <tg_id>")
		os.Exit(2)
	}
	if _, err := strconv.ParseInt(args[0], 10, 64); err != nil {
		fmt.Fprintf(os.Stderr, "bad tg_id %q: %v\n", args[0], err)
		os.Exit(2)
	}
	tgID := args[0]

	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		fmt.Fprintln(os.Stderr, "TELEGRAM_BOT_TOKEN is required in env")
		os.Exit(1)
	}

	const (
		title     = "Пионы 50 шт"
		ask       = int64(30000)
		offerLow  = int64(27000)
		offerHigh = int64(31500)
		buyer     = "Trust 2000"
		seller    = "Анна"
	)
	const photo = "" // без фото — тестим текстовые шаблоны
	const offerID = int64(99999)
	const bouquetID = int64(1)

	// Между сообщениями небольшая пауза, чтобы не упереться в rate-limit
	// бота (Telegram: 30 msg/sec на одного юзера — нам хватает с запасом,
	// но для гарантированного порядка прихода в чат лучше последовательно).
	send := func(label string, fn func()) {
		fmt.Printf("→ %s\n", label)
		fn()
		time.Sleep(400 * time.Millisecond)
	}

	send("1/8 NewOffer (со скидкой −10%)", func() {
		telegram.NotifyNewOffer(tgID, offerID, bouquetID, title, photo, ask, offerLow, buyer)
	})
	send("2/8 NewOffer (с надбавкой +5%)", func() {
		telegram.NotifyNewOffer(tgID, offerID+1, bouquetID, title, photo, ask, offerHigh, buyer)
	})
	send("3/8 OfferCountered", func() {
		telegram.NotifyOfferCountered(tgID, offerID+2, bouquetID, title, photo, offerLow, 28500, seller)
	})
	send("4/8 OfferAccepted (покупателю)", func() {
		telegram.NotifyOfferAccepted(tgID, title, offerLow, seller)
	})
	send("5/8 DealConfirmed (продавцу)", func() {
		telegram.NotifyDealConfirmed(tgID, title, offerLow)
	})
	send("6/8 OfferRejected", func() {
		telegram.NotifyOfferRejected(tgID, title, offerLow)
	})
	send("7/8 OfferWithdrawn (продавцу)", func() {
		telegram.NotifyOfferWithdrawn(tgID, title, offerLow)
	})
	send("8a/8 DealCancelled (получатель = продавец)", func() {
		telegram.NotifyDealCancelled(tgID, title, offerLow, false)
	})
	send("8b/8 DealCancelled (получатель = покупатель)", func() {
		telegram.NotifyDealCancelled(tgID, title, offerLow, true)
	})
	send("9/9 OfferExpired", func() {
		telegram.NotifyOfferExpired(tgID, title, offerLow)
	})

	fmt.Println("✓ done")
}
