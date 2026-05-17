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
		// Зеркальная нотификация продавцу-инициатору: подтверждение что
		// сделка реально оформилась + кнопка «Открыть Сделки» в DM.
		go telegram.NotifyDealConfirmed(ctx.SellerID, ctx.BouquetTitle, ctx.Price)
		// SSE-event покупателю — он узнает мгновенно даже сидя в мини-аппе
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferAccepted, OfferID: offerID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
		for _, id := range expired {
			go notifyExpired(db, id)
		}
		go adminbot.Record(db, adminbot.EventOfferAccepted,
			adminPayload(ctx),
			adminOfferText("✅ Сделка принята", ctx, ctx.Price, 0),
		)
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
		go adminbot.Record(db, adminbot.EventOfferRejected,
			adminPayload(ctx),
			adminOfferText("❌ Сделка отклонена", ctx, ctx.Price, 0),
		)
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
		go adminbot.Record(db, adminbot.EventOfferCountered,
			map[string]any{
				"old_offer_id": ctx.ID,
				"new_offer_id": newID,
				"bouquet_id":   ctx.BouquetID,
				"buyer_id":     ctx.BuyerID,
				"seller_id":    ctx.SellerID,
				"old_price":    ctx.Price,
				"new_price":    req.Price,
			},
			adminOfferText("🔄 Встречная цена", ctx, req.Price, ctx.Price),
		)
		response.OK(c, gin.H{"status": "countered", "new_offer_id": newID})

	case "cancel":
		cancelCtx, err := CancelAcceptedOffer(db, offerID, uid)
		if err != nil {
			serviceErr(c, err)
			return
		}
		// Уведомляем ПРОТИВОПОЛОЖНУЮ сторону. Кто инициатор — тот и не получает,
		// чтобы не было «вы сами отменили». recipientIsBuyer определяет тон и
		// CTA в нотификации (см. NotifyDealCancelled).
		var otherID string
		var recipientIsBuyer bool
		if uid == cancelCtx.BuyerID {
			// Инициатор — покупатель → получатель = продавец.
			otherID, recipientIsBuyer = cancelCtx.SellerID, false
		} else {
			// Инициатор — продавец → получатель = покупатель.
			otherID, recipientIsBuyer = cancelCtx.BuyerID, true
		}
		go telegram.NotifyDealCancelled(otherID, cancelCtx.BouquetTitle, cancelCtx.Price, recipientIsBuyer)
		events.Default().Publish(otherID, events.Event{
			Type: events.TypeOfferCancelled, OfferID: offerID,
			BouquetTitle: cancelCtx.BouquetTitle, Price: cancelCtx.Price,
		})
		initiator := "продавец"
		if uid == cancelCtx.BuyerID {
			initiator = "покупатель"
		}
		go adminbot.Record(db, adminbot.EventOfferCancelled,
			map[string]any{
				"offer_id":     cancelCtx.ID,
				"bouquet_id":   cancelCtx.BouquetID,
				"buyer_id":     cancelCtx.BuyerID,
				"seller_id":    cancelCtx.SellerID,
				"price":        cancelCtx.Price,
				"initiated_by": uid,
			},
			adminOfferText("⚠️ Сделка отменена (инициатор: "+initiator+")", cancelCtx, cancelCtx.Price, 0),
		)
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
		go adminbot.Record(db, adminbot.EventOfferWithdrawn,
			adminPayload(wCtx),
			adminOfferText("↩️ Предложение отозвано", wCtx, wCtx.Price, 0),
		)
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
	adminbot.Record(db, adminbot.EventOfferExpired,
		adminPayload(ctx),
		adminOfferText("⏱ Предложение истекло (опоздал на accept)", ctx, ctx.Price, 0),
	)
}

// adminPayload — общий map[string]any для всех action'ов respond.go и
// notifyExpired. Все поля стандартные, чтобы потом в админ-UI можно было
// фильтровать по любому identifier'у.
func adminPayload(ctx *OfferContext) map[string]any {
	return map[string]any{
		"offer_id":   ctx.ID,
		"bouquet_id": ctx.BouquetID,
		"buyer_id":   ctx.BuyerID,
		"seller_id":  ctx.SellerID,
		"price":      ctx.Price,
	}
}

// adminOfferText — общий шаблон админ-нотификации по offer-событию.
// Все 6 типов (accept/reject/counter/cancel/withdraw/expired) делят
// одну верстку: заголовок, букет, цена, опционально oldPrice/delta
// (для counter), кто покупатель, кто продавец.
//
// oldPrice = 0 → не показываем «было X ₸» (не имеет смысла для accept/reject/etc).
// Дельта от oldPrice к newPrice показывается только если oldPrice > 0.
func adminOfferText(header string, ctx *OfferContext, price, oldPrice int64) string {
	priceLine := fmt.Sprintf("Цена: <b>%s ₸</b>", adminbot.FormatPrice(price))
	if oldPrice > 0 {
		priceLine += fmt.Sprintf(" (было %s ₸%s)",
			adminbot.FormatPrice(oldPrice), deltaSuffix(oldPrice, price))
	}
	return fmt.Sprintf(
		"<b>%s</b>\n\n"+
			"<b>%s</b>\n"+
			"%s\n\n"+
			"Покупатель: %s\n"+
			"Продавец: %s",
		header,
		adminbot.EscapeHTML(ctx.BouquetTitle),
		priceLine,
		adminbot.FormatUser(ctx.BuyerID, ctx.BuyerName, ctx.BuyerUsername),
		adminbot.FormatUser(ctx.SellerID, ctx.SellerName, ctx.SellerUsername),
	)
}

// deltaSuffix — «(−15%)» / «(+3%)» или пусто (если разница <1%).
// Положительное значение pct = newPrice ниже ask (скидка, отрицательная
// для продавца). Используется и в create.go, и в respond.go.
func deltaSuffix(askPrice, offerPrice int64) string {
	if askPrice <= 0 {
		return ""
	}
	pct := int((askPrice - offerPrice) * 100 / askPrice)
	if pct == 0 {
		return ""
	}
	if pct > 0 {
		return fmt.Sprintf(" · −%d%%", pct)
	}
	return fmt.Sprintf(" · +%d%%", -pct)
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
