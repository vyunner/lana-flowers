package offers

import (
	"database/sql"
	"errors"
	"strings"
)

// Сервисный слой: бизнес-логика принять/отклонить/встречно.
// Используется и HTTP handler'ом, и webhook'ом telegram-бота —
// логика одна и та же, разные UI-обёртки.

// Возможные значения enum'ов в БД — собраны константами, чтобы исключить
// опечатки в SQL-строках и иметь grep'аемый список «вот так букет может быть».
const (
	OfferPending   = "pending"
	OfferAccepted  = "accepted"
	OfferRejected  = "rejected"
	OfferCountered = "countered"
	OfferCancelled = "cancelled"
	OfferExpired   = "expired"

	BouquetActive   = "active"
	BouquetSold     = "sold"
	BouquetArchived = "archived"
)

var (
	ErrOfferNotFound      = errors.New("offer not found")
	ErrNotSeller          = errors.New("not your offer (you are not the seller)")
	ErrOfferNotPending    = errors.New("offer is not pending")
	ErrInvalidPrice       = errors.New("price must be > 0")
	ErrSelfOffer          = errors.New("cannot offer on your own bouquet")
	ErrBouquetInactive    = errors.New("bouquet is not active")
	ErrBouquetNotFound    = errors.New("bouquet not found")
	ErrPendingExists      = errors.New("у вас уже есть активное предложение на этот букет")
	ErrCounterChainTooLong = errors.New("слишком длинная цепочка торга — примите или отклоните")
)

// MaxCounterDepth — сколько встречек подряд можно прислать. После этого
// бэк говорит «дальше уже только accept/reject», иначе цепочки разрастаются
// в бесконечную «ленту переписки» и засоряют БД.
const MaxCounterDepth = 10

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
	ParentID       sql.NullInt64
	Status         string
}

// loadContext — общая логика загрузки оффера. Принимает либо *sql.DB, либо
// *sql.Tx (через интерфейс querier). Если forUpdate=true, добавляется FOR
// UPDATE — блокировка строки оффера на время транзакции, защита от гонок
// при двух параллельных accept/counter/cancel.
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func loadContext(q querier, offerID int64, forUpdate bool) (*OfferContext, error) {
	o := &OfferContext{ID: offerID}
	var buyerLast, sellerLast string
	sqlText := `
		SELECT o.bouquet_id, o.buyer_id, o.seller_id, o.price, o.parent_id, o.status,
		       b.title,
		       COALESCE((b.photos)[1], '') AS bouquet_photo,
		       bu.first_name, bu.last_name, bu.username,
		       se.first_name, se.last_name, se.username
		FROM offers o
		JOIN bouquets b ON b.id = o.bouquet_id
		JOIN users bu ON bu.user_id = o.buyer_id
		JOIN users se ON se.user_id = o.seller_id
		WHERE o.id = $1`
	if forUpdate {
		// Только строку offers, не trying to lock JOIN'нутые users/bouquets.
		sqlText += ` FOR UPDATE OF o`
	}
	err := q.QueryRow(sqlText, offerID).Scan(
		&o.BouquetID, &o.BuyerID, &o.SellerID, &o.Price, &o.ParentID, &o.Status,
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
		o.BuyerName = strings.TrimSpace(o.BuyerName + " " + buyerLast)
	}
	if sellerLast != "" {
		o.SellerName = strings.TrimSpace(o.SellerName + " " + sellerLast)
	}
	return o, nil
}

// LoadContext — публичный read-only вариант. Не блокирует, читает свежий снапшот.
// Используется для нотификаций / отображения, где race не критична.
func LoadContext(db *sql.DB, offerID int64) (*OfferContext, error) {
	return loadContext(db, offerID, false)
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
//   1. блокирует строку оффера через FOR UPDATE (защита от двойного accept'а
//      когда параллельно прилетел запрос из mini-app и из бот-callback'а)
//   2. этот оффер → accepted
//   3. все ОСТАЛЬНЫЕ pending-офферы на этом букете → expired
//      (продавец не может «принять всех» — кто опоздал, тот получил отказ)
//   4. букет → sold
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

// RejectOffer — продавец отклоняет.
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
