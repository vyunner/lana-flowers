// Package telegram — минимальный HTTP-клиент к Bot API.
// Используется для рассылки уведомлений и обработки callback-ов из webhook'а.
//
// Структура пакета:
//   - client.go  (этот файл) — call() с retry + graceful-shutdown трекинг in-flight notify
//   - markup.go  — типы клавиатур (InlineKeyboard, ForceReply, ReplyKeyboard, WebAppInfo)
//   - methods.go — обёртки над Bot API методами (SendMessage, SendPhoto, EditMessage…)
//   - notify*.go — высокоуровневые Notify-функции, используемые из handler'ов
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const apiBase = "https://api.telegram.org/bot"

var httpClient = &http.Client{Timeout: 10 * time.Second}

func token() string { return os.Getenv("TELEGRAM_BOT_TOKEN") }

// ---- Graceful shutdown: трекинг in-flight notify-горутин ----

// pendingNotifs — счётчик in-flight notify-горутин. Все Notify*-функции
// inc'ают перед стартом своей работы (или вызовом в горутине) и
// dec'ают по завершению. WaitNotifications() в shutdown даёт им долететь.
var pendingNotifs sync.WaitGroup

// trackStart должна вызываться в начале каждой Notify*-функции, чтобы
// WaitNotifications мог их дождаться при graceful shutdown. Используется
// `defer trackEnd(trackStart())` сразу в начале функции.
func trackStart() struct{} {
	pendingNotifs.Add(1)
	return struct{}{}
}
func trackEnd(struct{}) { pendingNotifs.Done() }

// WaitNotifications ждёт окончания всех in-flight notify-горутин до
// timeout'а. Возвращает true если всё успели, false если по таймауту
// сдались (часть нотификаций потеряется при таком завершении).
func WaitNotifications(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		pendingNotifs.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// ---- HTTP call с retry+backoff ----

// Telegram изредка отдаёт 429/5xx — без retry'я мы теряем уведомления
// (notify-функции работают через `go ...` без очереди). Три попытки
// с 0/300/900мс — на нагрузку не повлияет, но поднимет deliverability
// до ~99% (по нашему опыту нативное окно сбоев у telegram-api — единицы секунд).
const maxRetries = 3

func call(method string, body any, out any) error {
	tk := token()
	if tk == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 300мс, 900мс — экспонента с базой 3.
			time.Sleep(time.Duration(attempt*attempt) * 300 * time.Millisecond)
		}

		req, err := http.NewRequest("POST", apiBase+tk+"/"+method, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // сетевая ошибка — retry
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 5xx и 429 — retry; всё остальное (4xx, 2xx) — финал, не пытаемся снова
		// потому что Telegram уже сказал «бизнесово неверно» (например невалидный chat_id).
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("telegram HTTP %d", resp.StatusCode)
			continue
		}

		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
		}
		return nil
	}
	return lastErr
}
