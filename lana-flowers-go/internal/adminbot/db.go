package adminbot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
)

// Все DB-операции пакета. Структуры тут максимально плоские, без
// промежуточных репозиториев — пакет маленький, для трёх таблиц.

// ---- admins ----

type Admin struct {
	TGID          string
	GrantedBy     sql.NullString
	Notifications map[string]bool // ключи — event types, true = enabled
}

// IsAdmin — есть ли запись в admins. Самый частый запрос (вызывается на
// каждый update от админ-бота).
func IsAdmin(db *sql.DB, tgID string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admins WHERE tg_id = $1)`, tgID).Scan(&exists)
	return exists, err
}

// GetAdmin — полный объект админа с настройками. Возвращает nil, nil если
// юзер не админ (для удобства caller'у без отдельной IsAdmin проверки).
func GetAdmin(db *sql.DB, tgID string) (*Admin, error) {
	var a Admin
	a.TGID = tgID
	var notifsRaw []byte
	err := db.QueryRow(`SELECT granted_by, notifications FROM admins WHERE tg_id = $1`, tgID).
		Scan(&a.GrantedBy, &notifsRaw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Notifications = parseNotifications(notifsRaw)
	return &a, nil
}

// parseNotifications — JSONB → map[string]bool. Хранится только
// отключённые: ключ с false-значением. Отсутствующий ключ = ON (default).
//
// При чтении НЕ домерживаем дефолты — caller получает map с явно false
// для disabled, ничего для enabled. EventEnabled() уже учитывает это.
func parseNotifications(raw []byte) map[string]bool {
	out := map[string]bool{}
	if len(raw) == 0 || string(raw) == "{}" {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// EventEnabled — включена ли нотификация типа eventType у этого админа.
// Логика: если в map ключ отсутствует — ON; если присутствует со
// значением — берём явное значение.
func (a *Admin) EventEnabled(eventType string) bool {
	if a.Notifications == nil {
		return true
	}
	v, present := a.Notifications[eventType]
	if !present {
		return true
	}
	return v
}

// AdminsForEvent — список tg_id всех админов, у которых нотификация
// типа eventType включена. Используется в notify.go при рассылке DM.
//
// Логика: SELECT всех админов, у которых notifications->>'event_x'
// либо отсутствует, либо не false. PostgreSQL: COALESCE(jsonb-cast).
func AdminsForEvent(db *sql.DB, eventType string) ([]string, error) {
	rows, err := db.Query(`
		SELECT tg_id FROM admins
		WHERE COALESCE((notifications->>$1)::bool, true) = true
	`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// ToggleEvent — переключает нотификацию (ON/OFF) для админа. Атомарно
// через jsonb_set: если был enabled (нет ключа или true) → ставим false;
// если был disabled (false) → ставим true.
func ToggleEvent(db *sql.DB, tgID, eventType string) (newEnabled bool, err error) {
	// Сначала читаем текущее состояние.
	a, err := GetAdmin(db, tgID)
	if err != nil || a == nil {
		return false, err
	}
	newEnabled = !a.EventEnabled(eventType)

	// jsonb_set с create_missing=true. Значение — json.Marshal(bool) даёт "true"/"false".
	_, err = db.Exec(`
		UPDATE admins
		SET notifications = jsonb_set(notifications, ARRAY[$2], $3::jsonb, true)
		WHERE tg_id = $1
	`, tgID, eventType, strconv.FormatBool(newEnabled))
	return newEnabled, err
}

// AddAdmin — выдать доступ. Идемпотентно (ON CONFLICT DO NOTHING).
func AddAdmin(db *sql.DB, tgID, grantedBy string) error {
	_, err := db.Exec(`
		INSERT INTO admins (tg_id, granted_by) VALUES ($1, $2)
		ON CONFLICT (tg_id) DO NOTHING
	`, tgID, grantedBy)
	return err
}

// ---- admin_requests ----

type Request struct {
	TGID        string
	DisplayName string
	Username    string
}

// AddRequest — записать что юзер тапнул /start в админ-боте. Идемпотентно
// (если уже есть — обновляем display_name/username из свежих TG-данных).
func AddRequest(db *sql.DB, tgID, displayName, username string) error {
	_, err := db.Exec(`
		INSERT INTO admin_requests (tg_id, display_name, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (tg_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			username = EXCLUDED.username
	`, tgID, displayName, username)
	return err
}

// RemoveRequest — удалить заявку (после grant'а или отказа). Не возвращает
// ошибку если строки не было — caller-у важен факт что её больше нет.
func RemoveRequest(db *sql.DB, tgID string) error {
	_, err := db.Exec(`DELETE FROM admin_requests WHERE tg_id = $1`, tgID)
	return err
}

// ListRequests — все pending-заявки на доступ. Используется в меню «Доступ».
func ListRequests(db *sql.DB) ([]Request, error) {
	rows, err := db.Query(`
		SELECT tg_id, display_name, username FROM admin_requests
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		if err := rows.Scan(&r.TGID, &r.DisplayName, &r.Username); err == nil {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

// ---- event_log ----

// LogEvent — append-only запись события. Payload — произвольный JSON
// (структура каждого event-type своя, см. вызовы adminbot.Record).
// Ошибки логируем но не возвращаем — caller (hookpoint в основном коде)
// не должен падать из-за лог-стораджа.
func LogEvent(db *sql.DB, eventType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("adminbot LogEvent marshal %s: %v\n", eventType, err)
		return
	}
	if _, err := db.Exec(
		`INSERT INTO event_log (event_type, payload) VALUES ($1, $2)`,
		eventType, raw,
	); err != nil {
		fmt.Printf("adminbot LogEvent insert %s: %v\n", eventType, err)
	}
}
