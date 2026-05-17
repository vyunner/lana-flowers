package offers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"lana-flowers-go/internal/adminbot"
	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/telegram"
)

// PendingTTL — через сколько pending-оффер автоматически expires если
// продавец ничего не ответил. 2 часа — флористика быстрая (букеты не
// должны висеть в торге днями, иначе клиент уйдёт к конкуренту или
// букет завянет), продавец либо отвечает в течение этого окна, либо
// оффер закрывается и покупатель ищет другое.
//
// Должно совпадать с OFFER_TTL_MS в lana-flowers-web/src/utils/format.js
// (фронт сам считает «истекает через …» от created_at — рассинхрон даст
// неправильный таймер в UI).
const PendingTTL = 2 * time.Hour

// expireSweepInterval — как часто крутить sweep'ер. 5 минут — иначе при
// TTL=2ч задержка sweep'а в 30 минут это 25% жизни оффера, фронт-таймер
// успевал показать «истёк» а статус ещё pending. С 5-минутным интервалом
// рассинхрон UI ≤ 5 минут — терпимо.
const expireSweepInterval = 5 * time.Minute

// StartPendingExpireWorker — фоновый goroutine, периодически закрывает
// pending-офферы старше PendingTTL. Запускается из main.go при старте
// сервиса. Останавливается через context cancellation.
//
// Реализация: в одной транзакции UPDATE... RETURNING id, buyer_id —
// атомарно получаем список заэкспайренных. После commit'а шлём DM/SSE
// в отдельной горутине, чтобы не блокировать sweep.
func StartPendingExpireWorker(ctx context.Context, db *sql.DB) {
	go func() {
		// Первый sweep сразу — на случай если сервер долго лежал.
		runExpireSweep(ctx, db)

		t := time.NewTicker(expireSweepInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runExpireSweep(ctx, db)
			}
		}
	}()
}

type expiredOffer struct {
	ID           int64
	BuyerID      string
	BouquetTitle string
	Price        int64
}

func runExpireSweep(ctx context.Context, db *sql.DB) {
	cutoff := time.Now().Add(-PendingTTL)
	rows, err := db.QueryContext(ctx, `
		WITH expired AS (
			UPDATE offers
			SET status = $1, responded_at = NOW()
			WHERE status = $2 AND created_at < $3
			RETURNING id, buyer_id, bouquet_id, price
		)
		SELECT e.id, e.buyer_id, b.title, e.price
		FROM expired e
		JOIN bouquets b ON b.id = e.bouquet_id
	`, OfferExpired, OfferPending, cutoff)
	if err != nil {
		log.Printf("expire-sweep query: %v", err)
		return
	}
	defer rows.Close()

	var expired []expiredOffer
	for rows.Next() {
		var e expiredOffer
		if scanErr := rows.Scan(&e.ID, &e.BuyerID, &e.BouquetTitle, &e.Price); scanErr == nil {
			expired = append(expired, e)
		}
	}
	if len(expired) == 0 {
		return
	}
	log.Printf("expire-sweep: expired %d pending offer(s)", len(expired))

	for _, e := range expired {
		go telegram.NotifyOfferExpired(e.BuyerID, e.BouquetTitle, e.Price)
		events.Default().Publish(e.BuyerID, events.Event{
			Type: events.TypeOfferExpired, OfferID: e.ID,
			BouquetTitle: e.BouquetTitle, Price: e.Price,
		})
		go adminbot.Record(db, adminbot.EventOfferExpired,
			map[string]any{
				"offer_id": e.ID,
				"buyer_id": e.BuyerID,
				"price":    e.Price,
				"reason":   "ttl_sweep",
			},
			fmt.Sprintf("⏱ <b>Предложение истекло (TTL)</b>\n%s — %d ₸",
				e.BouquetTitle, e.Price),
		)
	}
}
