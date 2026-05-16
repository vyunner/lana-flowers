// Package events — in-memory pub/sub шина по userID. Используется для
// SSE-стримов: handler'ы при изменении статуса оффера публикуют событие,
// все активные мини-апп клиенты получателя получают его мгновенно.
//
// Реализация специально простая: одна машина = одна шина, без Redis/NATS.
// Для горизонтального масштабирования (когда понадобится) подкладываем
// Redis pub/sub и подменяем имплементацию.
package events

import (
	"log"
	"sync"
)

// Event — что приходит подписчику. Минимум данных — клиент идёт по API
// за деталями (Deals список итд). Type — discriminator для toast'а.
type Event struct {
	Type         string `json:"type"`
	OfferID      int64  `json:"offer_id,omitempty"`
	BouquetTitle string `json:"bouquet_title,omitempty"`
	Price        int64  `json:"price,omitempty"`
}

// Возможные типы. Фронт ждёт ровно эти строки в Toast'е.
const (
	TypeOfferCreated   = "offer.created"   // юзеру (продавцу) пришёл новый оффер
	TypeOfferAccepted  = "offer.accepted"  // покупателю — приняли
	TypeOfferRejected  = "offer.rejected"  // покупателю — отклонили
	TypeOfferCountered = "offer.countered" // контрагенту — встречка прилетела
	TypeOfferCancelled = "offer.cancelled" // другой стороне — сделка отменена
	TypeOfferExpired   = "offer.expired"   // покупателю — букет ушёл другому
)

// Hub — потокобезопасный реестр подписчиков. Один подписчик = один
// open SSE-стрим. Один юзер может иметь несколько (открыто на двух
// устройствах) — всем шлём.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{} // userID → set of channels
}

func New() *Hub {
	return &Hub{subs: make(map[string]map[chan Event]struct{})}
}

// Subscribe возвращает канал событий и cleanup-функцию. Cleanup ОБЯЗАТЕЛЕН
// при завершении хендлера (через defer), иначе течёт горутина и memory.
func (h *Hub) Subscribe(userID string) (<-chan Event, func()) {
	ch := make(chan Event, 8) // small buffer чтобы один тормозной клиент не блокировал publish
	h.mu.Lock()
	if _, ok := h.subs[userID]; !ok {
		h.subs[userID] = make(map[chan Event]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()

	cleanup := func() {
		h.mu.Lock()
		if set, ok := h.subs[userID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, userID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cleanup
}

// Publish — не блокирует. Если канал переполнен (медленный клиент),
// событие дропается с warning'ом в лог. Лучше потерять одно событие
// чем стопорнуть весь handler.
func (h *Hub) Publish(userID string, e Event) {
	if userID == "" {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[userID] {
		select {
		case ch <- e:
		default:
			log.Printf("events: dropped %s for %s (subscriber buffer full)", e.Type, userID)
		}
	}
}

// defaultHub — singleton для удобства использования из handler'ов без
// прокидывания через все RegisterRoutes-сигнатуры. Тестам приходится
// мирится; OK для MVP.
var defaultHub = New()

// Default возвращает синглтон. Используется handler'ами как
// events.Default().Publish(...).
func Default() *Hub { return defaultHub }
