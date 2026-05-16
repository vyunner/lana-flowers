package offers

import (
	"database/sql"
	"errors"
	"net/http"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

type respondReq struct {
	Action string `json:"action" binding:"required"` // accept | reject | counter | cancel | withdraw
	Price  int64  `json:"price"`                     // обязательно для counter
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
		// SSE-event покупателю — он узнает мгновенно даже сидя в мини-аппе
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferAccepted, OfferID: offerID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
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
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferRejected, OfferID: offerID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
		response.OK(c, gin.H{"status": "rejected"})

	case "counter":
		newID, err := CounterOffer(db, offerID, uid, req.Price)
		if err != nil {
			serviceErr(c, err)
			return
		}
		go telegram.NotifyOfferCountered(
			ctx.BuyerID, newID, ctx.BouquetID, ctx.BouquetTitle, ctx.BouquetPhoto,
			ctx.Price, req.Price,
			ctx.SellerName,
		)
		// SSE — старому покупателю (он теперь responder на новый pending)
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferCountered, OfferID: newID,
			BouquetTitle: ctx.BouquetTitle, Price: req.Price,
		})
		response.OK(c, gin.H{"status": "countered", "new_offer_id": newID})

	case "cancel":
		cancelCtx, err := CancelAcceptedOffer(db, offerID, uid)
		if err != nil {
			serviceErr(c, err)
			return
		}
		// Уведомляем ПРОТИВОПОЛОЖНУЮ сторону. Кто инициатор — тот и не получает,
		// чтобы не было «вы сами отменили».
		var otherID string
		var iAmBuyer bool
		if uid == cancelCtx.BuyerID {
			otherID, iAmBuyer = cancelCtx.SellerID, true
		} else {
			otherID, iAmBuyer = cancelCtx.BuyerID, false
		}
		go telegram.NotifyDealCancelled(otherID, cancelCtx.BouquetTitle, cancelCtx.Price, iAmBuyer)
		events.Default().Publish(otherID, events.Event{
			Type: events.TypeOfferCancelled, OfferID: offerID,
			BouquetTitle: cancelCtx.BouquetTitle, Price: cancelCtx.Price,
		})
		response.OK(c, gin.H{"status": "cancelled"})

	case "withdraw":
		wCtx, err := WithdrawOwnPending(db, offerID, uid)
		if err != nil {
			serviceErr(c, err)
			return
		}
		// Уведомляем продавца — оффер исчез из его inbox'а.
		events.Default().Publish(wCtx.SellerID, events.Event{
			Type: events.TypeOfferCancelled, OfferID: offerID,
			BouquetTitle: wCtx.BouquetTitle, Price: wCtx.Price,
		})
		// В Telegram-DM продавцу тоже летим — иначе он не узнает, увидит
		// «пустоту» в inbox'е без объяснений. Используем именно Withdrawn,
		// а не DealCancelled: сделки не было, букет с продажи не уходил.
		go telegram.NotifyOfferWithdrawn(wCtx.SellerID, wCtx.BouquetTitle, wCtx.Price)
		response.OK(c, gin.H{"status": "withdrawn"})

	default:
		response.Err(c, http.StatusBadRequest, "BAD_ACTION",
			"action must be accept|reject|counter|cancel|withdraw")
	}
}

// notifyExpired — DM покупателю с заэкспайрившимся оффером после accept
// другого оффера на том же букете + SSE-event'ом если он сейчас сидит
// в мини-аппе.
func notifyExpired(db *sql.DB, offerID int64) {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return
	}
	telegram.NotifyOfferExpired(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
	events.Default().Publish(ctx.BuyerID, events.Event{
		Type: events.TypeOfferExpired, OfferID: offerID,
		BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
	})
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
	case errors.Is(err, ErrCounterChainTooLong):
		response.Err(c, http.StatusConflict, "COUNTER_CHAIN_TOO_LONG", err.Error())
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
