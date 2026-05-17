package offers

import (
	"database/sql"
	"strings"
)

// Read-only операции для loadContext'а и около него. Все мутации
// (accept/counter/cancel) внутри своих транзакций зовут loadContext
// с forUpdate=true; внешние caller'ы (handler'ы, нотификации) — через
// публичный LoadContext без блокировки.

// loadContext — общая логика загрузки оффера. Принимает либо *sql.DB,
// либо *sql.Tx (через интерфейс querier). Если forUpdate=true, добавляется
// FOR UPDATE — блокировка строки оффера на время транзакции, защита от
// гонок при двух параллельных accept/counter/cancel.
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

// LoadContext — публичный read-only вариант. Не блокирует, читает свежий
// снапшот. Используется для нотификаций и отображения, где race не критична.
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
