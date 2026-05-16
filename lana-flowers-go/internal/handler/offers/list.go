package offers

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

// OfferRich — оффер со всем что нужно для UI «Сделок»:
// данные букета, контрагента и (только для принятых) — телефон контрагента.
type OfferRich struct {
	ID          int64        `json:"id"`
	BouquetID   int64        `json:"bouquet_id"`
	Bouquet     BouquetMini  `json:"bouquet"`
	BuyerID     string       `json:"buyer_id"`
	SellerID    string       `json:"seller_id"`
	Price       int64        `json:"price"`
	Message     string       `json:"message"`
	Status      string       `json:"status"`
	ParentID    *int64       `json:"parent_id,omitempty"`
	CreatedAt   string       `json:"created_at"`
	RespondedAt string       `json:"responded_at,omitempty"`
	// Role — кем я являюсь в этом оффере: "buyer" | "seller".
	// Избавляет фронт от ручного сравнения с me.user_id.
	Role         string       `json:"role"`
	Counterparty Counterparty `json:"counterparty"`
}

type BouquetMini struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Price int64  `json:"price"` // цена объявления (для контекста)
	Photo string `json:"photo"` // url первой фотки или ""
}

// Counterparty — данные «второй стороны». Phone не пусто ТОЛЬКО для принятых
// сделок (status='accepted'). Это контракт приватности — телефон раскрывается
// только когда сделка состоялась.
type Counterparty struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// ListSent — мои отправленные офферы (я как покупатель в этом оффере).
//   GET /offers/sent
func ListSent(c *gin.Context, db *sql.DB) {
	listOffers(c, db, "buyer")
}

// ListReceived — входящие на меня (я как продавец в этом оффере).
//   GET /offers/received
func ListReceived(c *gin.Context, db *sql.DB) {
	listOffers(c, db, "seller")
}

// listOffers — общий код. role: "buyer" → ищем по o.buyer_id, "seller" → по o.seller_id.
// Counterparty подтягиваем из ПРОТИВОПОЛОЖНОГО поля.
func listOffers(c *gin.Context, db *sql.DB, role string) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)

	var myField, otherField string
	if role == "buyer" {
		myField, otherField = "o.buyer_id", "o.seller_id"
	} else {
		myField, otherField = "o.seller_id", "o.buyer_id"
	}

	statusFilter := c.Query("status")

	// JOIN'им букет (для title/photo) и контрагента (имя/аватар/телефон).
	// Phone отдаём ТОЛЬКО если status='accepted' — иначе пустая строка.
	q := `
		SELECT o.id, o.bouquet_id, o.buyer_id, o.seller_id, o.price, o.message,
		       o.status, o.parent_id, o.created_at,
		       COALESCE(o.responded_at::text, ''),
		       b.title, b.price,
		       COALESCE((b.photos)[1], '') AS bouquet_photo,
		       cp.user_id,
		       COALESCE(NULLIF(cp.display_name, ''), TRIM(CONCAT(cp.first_name, ' ', cp.last_name))) AS cp_name,
		       COALESCE(cp.avatar_url, ''),
		       CASE WHEN o.status = '` + OfferAccepted + `' THEN COALESCE(cp.phone_number, '') ELSE '' END AS cp_phone
		FROM offers o
		JOIN bouquets b ON b.id = o.bouquet_id
		JOIN users cp ON cp.user_id = ` + otherField + `
		WHERE ` + myField + ` = $1
	`
	args := []any{uid}
	if statusFilter != "" {
		q += ` AND o.status = $2`
		args = append(args, statusFilter)
	}
	q += ` ORDER BY o.created_at DESC`

	rows, err := db.Query(q, args...)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := []OfferRich{}
	for rows.Next() {
		var o OfferRich
		var parent sql.NullInt64
		if err := rows.Scan(
			&o.ID, &o.BouquetID, &o.BuyerID, &o.SellerID, &o.Price, &o.Message,
			&o.Status, &parent, &o.CreatedAt, &o.RespondedAt,
			&o.Bouquet.Title, &o.Bouquet.Price, &o.Bouquet.Photo,
			&o.Counterparty.UserID, &o.Counterparty.Name, &o.Counterparty.AvatarURL,
			&o.Counterparty.Phone,
		); err != nil {
			continue
		}
		o.Bouquet.ID = o.BouquetID
		if parent.Valid {
			v := parent.Int64
			o.ParentID = &v
		}
		o.Role = role
		out = append(out, o)
	}

	response.OK(c, out)
}
