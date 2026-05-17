package offers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"lana-flowers-go/internal/adminbot"
	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

type createReq struct {
	BouquetID int64 `json:"bouquet_id" binding:"required"`
	Price     int64 `json:"price" binding:"required"`
}

// Create — покупатель создаёт оффер на букет.
//   POST /offers   { bouquet_id, price }
func Create(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	buyerID := uidVal.(string)

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	id, ctx, err := CreateOffer(db, req.BouquetID, buyerID, req.Price)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPrice):
			response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		case errors.Is(err, ErrSelfOffer):
			response.Err(c, http.StatusBadRequest, "SELF_OFFER", err.Error())
		case errors.Is(err, ErrBouquetInactive), errors.Is(err, ErrBouquetNotFound):
			// Оба случая объединяем в один клиентский код — UX одинаков:
			// объявление больше нельзя купить (снято / продано / удалено).
			response.Err(c, http.StatusGone, "BOUQUET_UNAVAILABLE", err.Error())
		case errors.Is(err, ErrPendingExists):
			response.Err(c, http.StatusConflict, "DUPLICATE_OFFER", err.Error())
		default:
			response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		}
		return
	}

	// Уведомление продавцу — асинхронно, не блокируем ответ.
	if ctx != nil {
		sellerPrice, _ := GetSellerPrice(db, ctx.BouquetID)
		go telegram.NotifyNewOffer(
			ctx.SellerID,
			ctx.ID, ctx.BouquetID, ctx.BouquetTitle, ctx.BouquetPhoto,
			sellerPrice, ctx.Price,
			ctx.BuyerName,
		)
		// In-app push продавцу — если у него сейчас открыт мини-апп.
		events.Default().Publish(ctx.SellerID, events.Event{
			Type: events.TypeOfferCreated, OfferID: ctx.ID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
		go adminbot.Record(db, adminbot.EventOfferCreated,
			map[string]any{
				"offer_id":   ctx.ID,
				"bouquet_id": ctx.BouquetID,
				"buyer_id":   ctx.BuyerID,
				"seller_id":  ctx.SellerID,
				"price":      ctx.Price,
				"ask_price":  sellerPrice,
			},
			fmt.Sprintf("💌 <b>Новое предложение</b>\n%s — %d ₸ (ask: %d ₸)",
				ctx.BouquetTitle, ctx.Price, sellerPrice),
		)
	}

	response.OK(c, gin.H{"id": id})
}
