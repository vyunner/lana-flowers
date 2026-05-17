package adminbot

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Утилиты для форматирования админ-нотификаций — единый стиль во всех
// hookpoint'ах основного бэка (webhook/bouquets/offers).

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// EscapeHTML — минимальный escape для HTML parse_mode. Имена юзеров и
// названия букетов попадают в текст сообщения как user-content, могут
// содержать <>&.
func EscapeHTML(s string) string {
	return htmlEscaper.Replace(s)
}

// FormatPrice превращает 12500 → "12 500" с обычным пробелом. В чат-листе
// админа узких пузырей нет, NBSP не нужен (в отличие от telegram пакета).
func FormatPrice(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var sb strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		sb.WriteString(s[:pre])
		if len(s) > pre {
			sb.WriteByte(' ')
		}
	}
	for i := pre; i < len(s); i += 3 {
		sb.WriteString(s[i : i+3])
		if i+3 < len(s) {
			sb.WriteByte(' ')
		}
	}
	return sb.String()
}

// FormatUser — компактная человеко-читаемая строка про юзера для текста
// нотификации, всё в одну строку.
//
//	FormatUser("959568457", "Иван", "vyunner") →
//	  `<a href="tg://user?id=959568457">Иван</a> @vyunner`
//
// Имя завёрнуто в tg://user-ссылку — админ тапает и сразу открывается
// чат с юзером (даже если у того нет @username). Если name пустое —
// фолбэк на tg:<id>.
func FormatUser(tgID, name, username string) string {
	safeName := EscapeHTML(strings.TrimSpace(name))
	if safeName == "" {
		safeName = "tg:" + tgID
	}
	s := fmt.Sprintf(`<a href="tg://user?id=%s">%s</a>`, tgID, safeName)
	if username != "" {
		s += " @" + username
	}
	return s
}

// LookupUser — компактная инфа о юзере для админ-нотификаций.
// Используется в callsite'ах, где известен только tg_id и нужно
// отрендерить «Имя (@username) +77...». NotFound → пустые строки +
// nil error (caller всё равно отрисует читабельный fallback через FormatUser).
func LookupUser(db *sql.DB, tgID string) (name, username, phone string) {
	_ = db.QueryRow(`
		SELECT
			COALESCE(NULLIF(display_name, ''),
				TRIM(first_name || ' ' || last_name)) AS name,
			COALESCE(username, ''),
			COALESCE(phone_number, '')
		FROM users WHERE user_id = $1
	`, tgID).Scan(&name, &username, &phone)
	return
}
