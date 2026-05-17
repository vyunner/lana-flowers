package offers

import "database/sql"

// AcceptOffer — продавец (sellerID) принимает оффер.
//
// Транзакция:
//  1. блокирует строку оффера через FOR UPDATE (защита от двойного accept'а
//     когда параллельно прилетел запрос из mini-app и из бот-callback'а)
//  2. этот оффер → accepted
//  3. все ОСТАЛЬНЫЕ pending-офферы на этом букете → expired
//     (продавец не может «принять всех» — кто опоздал, тот получил отказ)
//  4. букет → sold
//
// Возвращает ID'шники expired-офферов — caller разошлёт по ним notify.
func AcceptOffer(db *sql.DB, offerID int64, sellerID string) (expired []int64, err error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ctx, err := loadContext(tx, offerID, true)
	if err != nil {
		return nil, err
	}
	if ctx.SellerID != sellerID {
		return nil, ErrNotSeller
	}
	if ctx.Status != OfferPending {
		return nil, ErrOfferNotPending
	}

	if _, err := tx.Exec(
		`UPDATE offers SET status = $2, responded_at = NOW() WHERE id = $1`,
		offerID, OfferAccepted,
	); err != nil {
		return nil, err
	}

	// Гасим остальные pending'и на этом букете (от других покупателей и
	// возможные параллельные счётчики). Собираем их id для нотификации.
	rows, err := tx.Query(`
		UPDATE offers SET status = $3, responded_at = NOW()
		WHERE bouquet_id = $1 AND status = $4 AND id != $2
		RETURNING id
	`, ctx.BouquetID, offerID, OfferExpired, OfferPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			expired = append(expired, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if _, err := tx.Exec(
		`UPDATE bouquets SET status = $2, updated_at = NOW() WHERE id = $1`,
		ctx.BouquetID, BouquetSold,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return expired, nil
}
