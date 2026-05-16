package offers

import (
	"database/sql"
	"errors"
	"net/http"

	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

type createReq struct {
	BouquetID int64  `json:"bouquet_id" binding:"required"`
	Price     int64  `json:"price" binding:"required"`
	Message   string `json:"message"`
}

// Create — покупатель создаёт оффер на букет.
//   POST /offers   { bouquet_id, price, message? }
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

	id, ctx, err := CreateOffer(db, req.BouquetID, buyerID, req.Price, req.Message)
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
			ctx.BuyerName, ctx.Message,
		)
	}

	response.OK(c, gin.H{"id": id})
}
