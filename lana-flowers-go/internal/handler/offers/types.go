// Package offers — HTTP-handler'ы и сервисный слой работы с офферами.
//
// HTTP-слой (Create / Respond / List* / Routes) живёт в файлах с
// соответствующими именами (create.go, respond.go, list.go, routes.go).
// Сервисный слой («что делать с оффером в БД») разнесён по операциям:
//
//   types.go    — этот файл; константы статусов, ошибки, OfferContext, querier
//   load.go     — loadContext / LoadContext / GetSellerPrice (чистые reads)
//   accept.go   — AcceptOffer
//   counter.go  — CreateOffer + CounterOffer (обе вставляют новый оффер)
//   cancel.go   — RejectOffer + WithdrawOwnPending + CancelAcceptedOffer
//                  (все три закрывают существующий оффер, отличия только в auth-чек'ах
//                  и в эффекте на статус букета)
package offers

import (
	"database/sql"
	"errors"
)

// Возможные значения enum'ов в БД — собраны константами, чтобы исключить
// опечатки в SQL-строках и иметь grep'аемый список «вот так оффер/букет
// может быть».
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
	ErrOfferNotFound       = errors.New("offer not found")
	ErrNotSeller           = errors.New("not your offer (you are not the seller)")
	ErrOfferNotPending     = errors.New("offer is not pending")
	ErrInvalidPrice        = errors.New("price must be > 0")
	ErrSelfOffer           = errors.New("cannot offer on your own bouquet")
	ErrBouquetInactive     = errors.New("bouquet is not active")
	ErrBouquetNotFound     = errors.New("bouquet not found")
	ErrPendingExists       = errors.New("у вас уже есть активное предложение на этот букет")
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

// querier — общая абстракция для *sql.DB и *sql.Tx. Используется в
// loadContext чтобы один и тот же SELECT работал и read-only (через DB),
// и внутри транзакции с FOR UPDATE (через Tx).
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
}
