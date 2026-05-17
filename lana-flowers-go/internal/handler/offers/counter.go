package offers

import "database/sql"

// CreateOffer и CounterOffer — обе ВСТАВЛЯЮТ новый оффер в БД, разница в
// предусловиях:
//
//   - CreateOffer: покупатель делает первый ход (нет parent_id, есть
//     проверки self-offer / дубликат pending).
//   - CounterOffer: продавец отвечает встречкой (parent_id ссылается на
//     предыдущий оффер, депт цепочки лимитирован MaxCounterDepth).

// CreateOffer — покупатель создаёт оффер на букет. Возвращает ID + контекст для уведомлений.
//
// Не вешаем FOR UPDATE на bouquet: status может измениться между read и
// INSERT (например, букет станет sold), но unique-индекс idx_offers_pending_unique
// + явный re-check status'а на бэке после INSERT — это перебор. Для гонки
// «букет архивировали ровно во время CreateOffer» — приемлемо иметь редкий
// orphan pending, который чистится auto-expire при следующем accept.
func CreateOffer(db *sql.DB, bouquetID int64, buyerID string, price int64) (int64, *OfferContext, error) {
	if price <= 0 {
		return 0, nil, ErrInvalidPrice
	}

	var sellerID, status string
	err := db.QueryRow(`SELECT seller_id, status FROM bouquets WHERE id = $1`, bouquetID).
		Scan(&sellerID, &status)
	if err == sql.ErrNoRows {
		return 0, nil, ErrBouquetNotFound
	}
	if err != nil {
		return 0, nil, err
	}
	if status != BouquetActive {
		return 0, nil, ErrBouquetInactive
	}
	if sellerID == buyerID {
		return 0, nil, ErrSelfOffer
	}

	// Дубликат: один pending-оффер на пару (bouquet, buyer).
	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM offers WHERE bouquet_id=$1 AND buyer_id=$2 AND status=$3)`,
		bouquetID, buyerID, OfferPending,
	).Scan(&exists); err != nil {
		return 0, nil, err
	}
	if exists {
		return 0, nil, ErrPendingExists
	}

	var id int64
	err = db.QueryRow(`
		INSERT INTO offers (bouquet_id, buyer_id, seller_id, price)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, bouquetID, buyerID, sellerID, price).Scan(&id)
	if err != nil {
		return 0, nil, err
	}

	ctx, err := LoadContext(db, id)
	return id, ctx, err
}

// CounterOffer — встречное предложение от продавца.
// Возвращает ID нового оффера (где buyer/seller теперь поменяны местами — ход у изначального покупателя).
func CounterOffer(db *sql.DB, offerID int64, sellerID string, newPrice int64) (int64, error) {
	if newPrice <= 0 {
		return 0, ErrInvalidPrice
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	ctx, err := loadContext(tx, offerID, true)
	if err != nil {
		return 0, err
	}
	if ctx.SellerID != sellerID {
		return 0, ErrNotSeller
	}
	if ctx.Status != OfferPending {
		return 0, ErrOfferNotPending
	}

	// Лимит глубины counter-chain — считаем длину цепочки через parent_id
	// до корневого оффера. Без лимита юзеры могут бесконечно торговаться
	// (а каждый шаг — это INSERT с join'ами).
	var depth int
	if err := tx.QueryRow(`
		WITH RECURSIVE chain AS (
			SELECT id, parent_id FROM offers WHERE id = $1
			UNION ALL
			SELECT o.id, o.parent_id FROM offers o JOIN chain c ON o.id = c.parent_id
		)
		SELECT COUNT(*) FROM chain
	`, offerID).Scan(&depth); err != nil {
		return 0, err
	}
	if depth >= MaxCounterDepth {
		return 0, ErrCounterChainTooLong
	}

	// Закрываем ВСЕ pending-офферы между этой парой (buyer↔seller) на этом букете,
	// не только явный offerID. Иначе если параллельно висит ещё один pending
	// (например, оригинальный покупатель создал свежий оффер пока шёл counter-обмен),
	// INSERT ниже упадёт на idx_offers_pending_unique.
	if _, err := tx.Exec(`
		UPDATE offers SET status = $4, responded_at = NOW()
		WHERE bouquet_id = $1 AND status = $5
		  AND ((buyer_id = $2 AND seller_id = $3) OR (buyer_id = $3 AND seller_id = $2))
	`, ctx.BouquetID, ctx.BuyerID, ctx.SellerID, OfferCountered, OfferPending); err != nil {
		return 0, err
	}

	var newID int64
	if err := tx.QueryRow(`
		INSERT INTO offers (bouquet_id, buyer_id, seller_id, price, parent_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, ctx.BouquetID, ctx.SellerID, ctx.BuyerID, newPrice, offerID).Scan(&newID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newID, nil
}
