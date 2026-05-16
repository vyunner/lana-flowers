package offers

import (
	"database/sql"
	"errors"
	"fmt"
)

// Сервисный слой: бизнес-логика принять/отклонить/встречно.
// Используется и HTTP handler'ом, и webhook'ом telegram-бота —
// логика одна и та же, разные UI-обёртки.

var (
	ErrOfferNotFound   = errors.New("offer not found")
	ErrNotSeller       = errors.New("not your offer (you are not the seller)")
	ErrOfferNotPending = errors.New("offer is not pending")
	ErrInvalidPrice    = errors.New("price must be > 0")
	ErrSelfOffer       = errors.New("cannot offer on your own bouquet")
	ErrBouquetInactive = errors.New("bouquet is not active")
	ErrPendingExists   = errors.New("у вас уже есть активное предложение на этот букет")
)

// OfferContext — данные оффера, которые нужны и handler'у, и notify-функциям.
type OfferContext struct {
	ID             int64
	BouquetID      int64
	BouquetTitle   string
	BouquetPhoto   string // url первой фотки букета (для превью в нотификации)
	BuyerID        string
	BuyerName      string // first_name + last_name
	BuyerUsername  string
	SellerID       string
	SellerName     string
	SellerUsername string
	Price          int64
	Message        string
	ParentID       sql.NullInt64
	Status         string
}

// LoadContext — подгружает оффер с join на bouquets/users (для уведомлений).
func LoadContext(db *sql.DB, offerID int64) (*OfferContext, error) {
	o := &OfferContext{ID: offerID}
	var buyerLast, sellerLast string
	err := db.QueryRow(`
		SELECT o.bouquet_id, o.buyer_id, o.seller_id, o.price, o.message, o.parent_id, o.status,
		       b.title,
		       COALESCE((b.photos)[1], '') AS bouquet_photo,
		       bu.first_name, bu.last_name, bu.username,
		       se.first_name, se.last_name, se.username
		FROM offers o
		JOIN bouquets b ON b.id = o.bouquet_id
		JOIN users bu ON bu.user_id = o.buyer_id
		JOIN users se ON se.user_id = o.seller_id
		WHERE o.id = $1
	`, offerID).Scan(
		&o.BouquetID, &o.BuyerID, &o.SellerID, &o.Price, &o.Message, &o.ParentID, &o.Status,
		&o.BouquetTitle,
		&o.BouquetPhoto,
		&o.BuyerName, &buyerLast, &o.BuyerUsername,
		&o.SellerName, &sellerLast, &o.SellerUsername,
	)
	if err == sql.ErrNoRows {
		return nil, ErrOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	if buyerLast != "" {
		o.BuyerName = trimSpace(o.BuyerName + " " + buyerLast)
	}
	if sellerLast != "" {
		o.SellerName = trimSpace(o.SellerName + " " + sellerLast)
	}
	return o, nil
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

// GetSellerPrice достаёт цену букета (для отображения в нотификации).
func GetSellerPrice(db *sql.DB, bouquetID int64) (int64, error) {
	var price int64
	err := db.QueryRow(`SELECT price FROM bouquets WHERE id = $1`, bouquetID).Scan(&price)
	if err != nil {
		return 0, err
	}
	return price, nil
}

// AcceptOffer — продавец (sellerID) принимает оффер.
//
// Транзакция:
//   1. этот оффер → accepted
//   2. все ОСТАЛЬНЫЕ pending-офферы на этом букете → expired
//      (продавец не может «принять всех» — кто опоздал, тот получил отказ)
//   3. букет → sold
//
// Возвращает ID'шники expired-офферов — caller разошлёт по ним notify.
func AcceptOffer(db *sql.DB, offerID int64, sellerID string) (expired []int64, err error) {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return nil, err
	}
	if ctx.SellerID != sellerID {
		return nil, ErrNotSeller
	}
	if ctx.Status != "pending" {
		return nil, ErrOfferNotPending
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE offers SET status = 'accepted', responded_at = NOW() WHERE id = $1`, offerID); err != nil {
		return nil, err
	}

	// Гасим остальные pending'и на этом букете (от других покупателей и
	// возможные параллельные счётчики). Собираем их id для нотификации.
	rows, err := tx.Query(`
		UPDATE offers SET status = 'expired', responded_at = NOW()
		WHERE bouquet_id = $1 AND status = 'pending' AND id != $2
		RETURNING id
	`, ctx.BouquetID, offerID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			expired = append(expired, id)
		}
	}
	rows.Close()

	if _, err := tx.Exec(`UPDATE bouquets SET status = 'sold', updated_at = NOW() WHERE id = $1`, ctx.BouquetID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return expired, nil
}

// CancelAcceptedOffer — «сделка не состоялась». Может вызвать любая сторона
// (buyer или seller) на оффере в статусе accepted. Возвращает букет в активные.
//
// Другая сторона получает DM-уведомление об отмене (caller разошлёт через ctx).
// Возвращает контекст оффера — нужен для уведомления.
func CancelAcceptedOffer(db *sql.DB, offerID int64, userID string) (*OfferContext, error) {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return nil, err
	}
	if ctx.BuyerID != userID && ctx.SellerID != userID {
		return nil, ErrNotSeller // переиспользуем — суть та же: «вы не сторона сделки»
	}
	if ctx.Status != "accepted" {
		return nil, ErrOfferNotPending // «не в нужном статусе»
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE offers SET status = 'cancelled', responded_at = NOW() WHERE id = $1`, offerID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE bouquets SET status = 'active', updated_at = NOW() WHERE id = $1`, ctx.BouquetID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ctx, nil
}

// RejectOffer — продавец отклоняет.
func RejectOffer(db *sql.DB, offerID int64, sellerID string) error {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return err
	}
	if ctx.SellerID != sellerID {
		return ErrNotSeller
	}
	if ctx.Status != "pending" {
		return ErrOfferNotPending
	}
	_, err = db.Exec(`UPDATE offers SET status = 'rejected', responded_at = NOW() WHERE id = $1`, offerID)
	return err
}

// CounterOffer — встречное предложение от продавца.
// Возвращает ID нового оффера (где buyer/seller теперь поменяны местами — ход у изначального покупателя).
func CounterOffer(db *sql.DB, offerID int64, sellerID string, newPrice int64, message string) (int64, error) {
	ctx, err := LoadContext(db, offerID)
	if err != nil {
		return 0, err
	}
	if ctx.SellerID != sellerID {
		return 0, ErrNotSeller
	}
	if ctx.Status != "pending" {
		return 0, ErrOfferNotPending
	}
	if newPrice <= 0 {
		return 0, ErrInvalidPrice
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Закрываем ВСЕ pending-офферы между этой парой (buyer↔seller) на этом букете,
	// не только явный offerID. Иначе если параллельно висит ещё один pending
	// (например, оригинальный покупатель создал свежий оффер пока шёл counter-обмен),
	// INSERT ниже упадёт на idx_offers_pending_unique.
	if _, err := tx.Exec(`
		UPDATE offers SET status = 'countered', responded_at = NOW()
		WHERE bouquet_id = $1 AND status = 'pending'
		  AND ((buyer_id = $2 AND seller_id = $3) OR (buyer_id = $3 AND seller_id = $2))
	`, ctx.BouquetID, ctx.BuyerID, ctx.SellerID); err != nil {
		return 0, err
	}

	var newID int64
	if err := tx.QueryRow(`
		INSERT INTO offers (bouquet_id, buyer_id, seller_id, price, message, parent_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, ctx.BouquetID, ctx.SellerID, ctx.BuyerID, newPrice, message, offerID).Scan(&newID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newID, nil
}

// CreateOffer — покупатель создаёт оффер на букет. Возвращает ID + контекст для уведомлений.
func CreateOffer(db *sql.DB, bouquetID int64, buyerID string, price int64, message string) (int64, *OfferContext, error) {
	if price <= 0 {
		return 0, nil, ErrInvalidPrice
	}

	var sellerID, status string
	err := db.QueryRow(`SELECT seller_id, status FROM bouquets WHERE id = $1`, bouquetID).
		Scan(&sellerID, &status)
	if err == sql.ErrNoRows {
		return 0, nil, fmt.Errorf("bouquet not found")
	}
	if err != nil {
		return 0, nil, err
	}
	if status != "active" {
		return 0, nil, ErrBouquetInactive
	}
	if sellerID == buyerID {
		return 0, nil, ErrSelfOffer
	}

	// Дубликат: один pending-оффер на пару (bouquet, buyer).
	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM offers WHERE bouquet_id=$1 AND buyer_id=$2 AND status='pending')`,
		bouquetID, buyerID,
	).Scan(&exists); err != nil {
		return 0, nil, err
	}
	if exists {
		return 0, nil, ErrPendingExists
	}

	var id int64
	err = db.QueryRow(`
		INSERT INTO offers (bouquet_id, buyer_id, seller_id, price, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, bouquetID, buyerID, sellerID, price, message).Scan(&id)
	if err != nil {
		return 0, nil, err
	}

	ctx, err := LoadContext(db, id)
	return id, ctx, err
}
