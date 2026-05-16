package bouquets

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

// Delete — мягкое снятие объявления (статус → archived). Только владелец.
// Параллельно експайрит все pending-офферы на этом букете и пушит покупателям
// SSE + Telegram-DM «букет снят». Иначе их офферы висели бы вечно.
//   DELETE /bouquets/:id
func Delete(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)
	id := c.Param("id")
	ctx := c.Request.Context()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE bouquets SET status = $3, updated_at = NOW()
		WHERE id = $1 AND seller_id = $2 AND status != $3
	`, id, uid, offers.BouquetArchived)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "bouquet not found or not yours")
		return
	}

	// Экспайрим все pending'и → возвращаем id'шники для нотификации.
	expRows, err := tx.QueryContext(ctx, `
		UPDATE offers SET status = $2, responded_at = NOW()
		WHERE bouquet_id = $1 AND status = $3
		RETURNING id, buyer_id
	`, id, offers.OfferExpired, offers.OfferPending)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	type expiredRef struct {
		ID      int64
		BuyerID string
	}
	var expired []expiredRef
	for expRows.Next() {
		var e expiredRef
		if scanErr := expRows.Scan(&e.ID, &e.BuyerID); scanErr == nil {
			expired = append(expired, e)
		}
	}
	expRows.Close()

	if err := tx.Commit(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Нотификации после commit — чтобы не отправить и потом откатить tx.
	for _, e := range expired {
		ec, _ := offers.LoadContext(db, e.ID)
		if ec == nil {
			continue
		}
		go telegram.NotifyOfferExpired(ec.BuyerID, ec.BouquetTitle, ec.Price)
		events.Default().Publish(e.BuyerID, events.Event{
			Type: events.TypeOfferExpired, OfferID: e.ID,
			BouquetTitle: ec.BouquetTitle, Price: ec.Price,
		})
	}

	response.OK(c, gin.H{"archived": true, "expired_offers": len(expired)})
}
