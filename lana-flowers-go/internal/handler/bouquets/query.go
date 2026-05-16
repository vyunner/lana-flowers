package bouquets

import (
	"database/sql"

	"lana-flowers-go/internal/handler/offers"

	"github.com/lib/pq"
)

// bouquetWithSellerSQL — общий SELECT для list.go и get.go.
// Возвращает букет + denormalised поля продавца + LEFT JOIN на pending-оффер
// текущего юзера (для бейджа «Предложено» в карточке).
//
// Параметры по позициям:
//   $1 — current_user_id (для LEFT JOIN'а my_offer; "" если аноним)
// Дальше caller добавляет WHERE-условия с $2, $3 и т.д.
const bouquetWithSellerSelect = `
SELECT b.id, b.seller_id, b.title, b.description, b.price, b.city,
       b.photos, b.category, b.status, b.created_at,
       COALESCE(NULLIF(u.display_name, ''), u.first_name) AS seller_name,
       u.avatar_url,
       o.id AS my_offer_id,
       o.price AS my_offer_price
FROM bouquets b
JOIN users u ON u.user_id = b.seller_id
LEFT JOIN offers o
  ON o.bouquet_id = b.id
 AND o.buyer_id = $1
 AND o.status = '` + offers.OfferPending + `'`

// scanBouquetWithSeller — общий Scan для строки bouquetWithSellerSelect.
// Используется и в QueryRow, и в Query+Rows.Scan циклах.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanBouquetWithSeller(s rowScanner) (Bouquet, error) {
	var b Bouquet
	var seller SellerInfo
	var createdAt string
	var myOfferID sql.NullInt64
	var myOfferPrice sql.NullInt64

	if err := s.Scan(
		&b.ID, &b.SellerID, &b.Title, &b.Description, &b.Price, &b.City,
		pq.Array(&b.Photos), &b.Category, &b.Status, &createdAt,
		&seller.DisplayName, &seller.AvatarURL,
		&myOfferID, &myOfferPrice,
	); err != nil {
		return Bouquet{}, err
	}

	seller.UserID = b.SellerID
	b.Seller = &seller
	b.CreatedAt = createdAt
	if myOfferID.Valid {
		b.MyOffer = &MyOfferInfo{ID: myOfferID.Int64, Price: myOfferPrice.Int64}
	}
	return b, nil
}
