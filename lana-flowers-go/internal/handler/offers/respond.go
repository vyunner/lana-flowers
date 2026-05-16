package offers

import (
	"database/sql"
	"errors"
	"net/http"

	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

type respondReq struct {
	Action  string `json:"action" binding:"required"` // accept | reject | counter | cancel
	Price   int64  `json:"price"`                     // обязательно для counter
	Message string `json:"message"`
}

// Respond — продавец отвечает на оффер: принять / отклонить / встречно / отменить-сделку.
//   POST /offers/:id/respond
func Respond(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)

	var req respondReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	offerID := parseInt64(c.Param("id"))
	if offerID == 0 {
		response.Err(c, http.StatusBadRequest, "BAD_ID", "invalid offer id")
		return
	}

	ctx, err := LoadContext(db, offerID)
	if err != nil {
		serviceErr(c, err)
		return
	}

	switch req.Action {
	case "accept":
		expired, err := AcceptOffer(db, offerID, uid)
		if err != nil {
			serviceErr(c, err)
			return
		}
		go telegram.NotifyOfferAccepted(ctx.BuyerID, ctx.BouquetTitle, ctx.Price, ctx.SellerName)
		// Уведомляем тех у кого pending'и заэкспайрились (другие покупатели).
		for _, id := range expired {
			go notifyExpired(db, id)
		}
		response.OK(c, gin.H{"status": "accepted"})

	case "reject":
		if err := RejectOffer(db, offerID, uid); err != nil {
			serviceErr(c, err)
			return
		}
		go telegram.NotifyOfferRejected(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
		response.OK(c, gin.H{"status": "rejected"})

	case "counter":
		newID, err := CounterOffer(db, offerID, uid, req.Price, req.Message)
		if err != nil {
			serviceErr(c, err)
			return
		}
		go telegram.NotifyOfferCountered(
			ctx.BuyerID, newID, ctx.BouquetID, ctx.BouquetTitle, ctx.BouquetPhoto,
			ctx.Price, req.Price,
			ctx.SellerName, req.Message,
		)
		response.OK(c, gin.H{"status": "countered", "new_offer_id": newID})

	case "cancel":
		cancelCtx, err := CancelAcceptedOffer(db, offerID, uid)
		if err != nil {
			serviceErr(c, err)
			return
		}
		// Уведомляем ПРОТИВОПОЛОЖНУЮ сторону. Кто инициатор — тот и не получает,
		// чтобы не было «вы сами отменили».
		var otherID, otherName string
		var iAmBuyer bool
		if uid == cancelCtx.BuyerID {
			otherID, otherName, iAmBuyer = cancelCtx.SellerID, cancelCtx.SellerName, true
		} else {
			otherID, otherName, iAmBuyer = cancelCtx.BuyerID, cancelCtx.BuyerName, false
		}
		_ = otherName // имя пока не нужно для текста, может пригодиться позже
		go telegram.NotifyDealCancelled(otherID, cancelCtx.BouquetTitle, cancelCtx.Price, iAmBuyer)
		response.OK(c, gin.H{"status": "cancelled"})

	default:
		response.Err(c, http.StatusBadRequest, "BAD_ACTION", "action must be accept|reject|counter|cancel")
	}
}

// notifyExpired — DM покупателю с заэкспайрившимся оффером после accept
// другого оффера на том же букете. Подгружаем context отдельно: основной
// AcceptOffer возвращает только id-шники.
func notifyExpired(db *sql.DB, offerID int64) {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return
	}
	telegram.NotifyOfferExpired(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
}

func serviceErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrOfferNotFound):
		response.Err(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrNotSeller):
		response.Err(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, ErrOfferNotPending):
		response.Err(c, http.StatusBadRequest, "OFFER_NOT_PENDING", err.Error())
	case errors.Is(err, ErrInvalidPrice):
		response.Err(c, http.StatusBadRequest, "BAD_PRICE", err.Error())
	default:
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
	}
}

func parseInt64(s string) int64 {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int64(ch-'0')
	}
	return n
}
