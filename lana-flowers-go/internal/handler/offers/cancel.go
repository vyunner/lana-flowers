package offers

import "database/sql"

// Три варианта «закрыть оффер без принятия». Отличия только в auth и
// в эффекте на статус букета:
//
//   - RejectOffer            (seller на pending) — оффер→rejected, букет НЕ трогается
//   - WithdrawOwnPending     (buyer на pending)  — оффер→cancelled, букет НЕ трогается
//   - CancelAcceptedOffer    (любая сторона на accepted) — оффер→cancelled, букет→active
//
// Семантически все три — отказ от сделки, поэтому в одном файле.

// RejectOffer — продавец отклоняет pending-оффер. Букет НЕ трогаем —
// другие покупатели могут продолжать предлагать.
func RejectOffer(db *sql.DB, offerID int64, sellerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ctx, err := loadContext(tx, offerID, true)
	if err != nil {
		return err
	}
	if ctx.SellerID != sellerID {
		return ErrNotSeller
	}
	if ctx.Status != OfferPending {
		return ErrOfferNotPending
	}
	if _, err := tx.Exec(
		`UPDATE offers SET status = $2, responded_at = NOW() WHERE id = $1`,
		offerID, OfferRejected,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// WithdrawOwnPending — покупатель забирает СВОЙ pending-оффер (передумал).
// Доступно только покупателю (buyer_id == userID) и только пока pending.
// Букет НЕ трогаем — он остался активным, другие покупатели могут предлагать.
func WithdrawOwnPending(db *sql.DB, offerID int64, userID string) (*OfferContext, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ctx, err := loadContext(tx, offerID, true)
	if err != nil {
		return nil, err
	}
	if ctx.BuyerID != userID {
		return nil, ErrNotSeller // переиспользуем «вы не сторона сделки»
	}
	if ctx.Status != OfferPending {
		return nil, ErrOfferNotPending
	}
	if _, err := tx.Exec(
		`UPDATE offers SET status = $2, responded_at = NOW() WHERE id = $1`,
		offerID, OfferCancelled,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ctx, nil
}

// CancelAcceptedOffer — «сделка не состоялась». Может вызвать любая сторона
// (buyer или seller) на оффере в статусе accepted. Возвращает букет в активные.
//
// Другая сторона получает DM-уведомление об отмене (caller разошлёт через ctx).
// Возвращает контекст оффера — нужен для уведомления.
func CancelAcceptedOffer(db *sql.DB, offerID int64, userID string) (*OfferContext, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ctx, err := loadContext(tx, offerID, true)
	if err != nil {
		return nil, err
	}
	if ctx.BuyerID != userID && ctx.SellerID != userID {
		return nil, ErrNotSeller // «вы не сторона сделки»
	}
	if ctx.Status != OfferAccepted {
		return nil, ErrOfferNotPending // «не в нужном статусе»
	}

	if _, err := tx.Exec(
		`UPDATE offers SET status = $2, responded_at = NOW() WHERE id = $1`,
		offerID, OfferCancelled,
	); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(
		`UPDATE bouquets SET status = $2, updated_at = NOW() WHERE id = $1`,
		ctx.BouquetID, BouquetActive,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ctx, nil
}
