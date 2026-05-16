# Lana Flowers — Telegram Mini App для торга букетами

Монорепо: Go-бэк + Vue 3-фронт + Postgres. Mini App `@lana_flowers_bot`.
C2C-маркетплейс: продавцы публикуют букеты, покупатели предлагают цену,
ведут торг с встречными предложениями. Целевая аудитория — мужчины
22-45 в Казахстане, покупающие подарки.

## Структура

```
/
├── lana-flowers-go/         # backend (Gin + Postgres)
│   ├── cmd/api/             # entry: main + migrate cmd + thumbs cmd
│   ├── internal/
│   │   ├── auth/            # HMAC валидация initData (+ table tests)
│   │   ├── db/              # Postgres pool
│   │   ├── events/          # in-memory pub/sub hub для SSE
│   │   ├── image/           # thumbnail генерация (stdlib + x/image/draw)
│   │   ├── response/        # response.OK/Err хелперы
│   │   ├── telegram/        # bot API клиент + notify-функции (retry + wait-group)
│   │   └── handler/<domain>/  # bouquets / offers / users / upload / webhook / events
│   └── migrations/          # golang-migrate, текущая версия — 7
├── lana-flowers-web/        # frontend (Vue 3 + Vite, БЕЗ TypeScript)
│   └── src/
│       ├── api/             # тонкий fetch-клиент к бэку
│       ├── composables/     # useApi, usePolling, useEventStream, usePullToRefresh
│       ├── components/      # обычные компоненты
│       ├── components/base/ # BaseModal, BaseSheet, StatusState, PullToRefreshScroll, Toast
│       ├── state/           # глобальные ref'ы (auth, toasts, realtime)
│       ├── utils/           # format, dialog (tg.showAlert обёртки), image
│       └── data/            # constants (cities, categories)
└── deploy.sh                # git pull + go build + systemctl restart на VPS
```

## Production

- VPS: `64.226.107.161` (root), `lana-api` systemd-юнит, путь `/root/lana-flowers/`
- TLS через Caddy + `64-226-107-161.nip.io` (TODO: купить нормальный домен)
- Postgres локально на VPS, бэкап БД пока НЕ настроен (TODO)
- Frontend: Vercel, prod alias `lana-flowers.vercel.app`
- Деплой бэка: `ssh root@64.226.107.161 'cd /root/lana-flowers && ./deploy.sh'`
- Деплой фронта: `cd lana-flowers-web && npx vercel --prod --yes`

## Архитектура

### Auth
- Все запросы (кроме `/health` и `/tg/webhook`) — `X-Telegram-Init-Data` header
- HMAC валидация по bot token, `initDataMaxAge=3h`
- SSE-endpoint берёт initData из `?tma=` query (EventSource не даёт headers)
- Phone-gate: юзер без `phone_number` блокируется на всех endpoint'ах
  кроме `/users/me` и `/upload` (онбординг сначала фото/имя, потом телефон)

### Realtime (SSE)
- `GET /events` — long-lived поток, ping каждые 25с
- При offer.* событиях бэк публикует через `events.Default().Publish()`
- Фронт слушает через `useEventStream`, рисует Toast'ы, рефрешит экраны
- Polling 60с в Deals/Profile/BouquetGrid — safety net на случай падения SSE
- Когда SSE подключён — polling в паузе через `state/realtime.sseConnected`

### Bargaining flow
- buyer creates offer (`POST /offers`) → seller получает DM в боте + SSE
- seller: Accept/Reject/Counter (callback_data) либо через mini-app
- Counter chain ограничен `MaxCounterDepth=10`
- pending-офферы старше 7 дней автоматически expired через background sweep
- accepted сделка может быть отменена любой стороной (action: cancel)
- buyer может отозвать свой pending (action: withdraw)
- При accept всех остальных pending'ов на букете → expired + DM/SSE покупателям

### Photos
- Upload: MIME-sniff через `http.DetectContentType`, файлы в `/uploads/<user_id>/<random>.<ext>`
- В bouquet.photos[] валидируется что URL начинается с `/uploads/`
- При upload автоматом генерится `<name>_thumb.jpg` (400px JPEG q=70) для каталога
- Frontend client-side resize фото до 1920px JPEG q=0.85 ДО upload'а
- Для регенерации thumb'ов после изменения логики: `./app thumbs /var/lib/lana-flowers/uploads`

### UI-конвенции
- BaseSheet / BaseModal обёрнуты в `<Teleport to="body">`, имеют `level`-prop
  для z-index стеков (CitySheet level=2 для открытия из SellSheet)
- `tg.showConfirm` / `tg.showAlert` через `utils/dialog.js` — никогда не `window.confirm`/`alert`
- `formatPrice(n)` / `formatPriceKzt(n)` из `utils/format.js` — единое форматирование
- `thumbUrl(originalUrl)` для миниатюр в листингах, оригинал в BouquetDetail
- Telegram BackButton автоматически hooked в BaseSheet/BaseModal/BouquetDetail
- Pull-to-refresh: `<PullToRefreshScroll :loader="fn">` (ждёт promise, min 400ms спиннер)

## Команды

```bash
# Backend локально
cd lana-flowers-go
cp .env.example .env  # заполнить DATABASE_URL, TELEGRAM_BOT_TOKEN, TELEGRAM_WEBHOOK_SECRET
go run ./cmd/api migrate up
go run ./cmd/api
# Тесты
go test ./internal/auth/...  # есть HMAC table tests, другие — пусто

# Frontend локально
cd lana-flowers-web
npm install
npm run dev

# Деплой
ssh root@64.226.107.161 'cd /root/lana-flowers && ./deploy.sh'
cd lana-flowers-web && npx vercel --prod --yes

# Миграции
ssh root@64.226.107.161 'cd /root/lana-flowers/lana-flowers-go && ./app migrate up'
```

## Env vars

```
APP_ENV=production           # включает gin release mode + требует webhook secret
DATABASE_URL=postgres://...
TELEGRAM_BOT_TOKEN=...
TELEGRAM_WEBHOOK_SECRET=...  # обязателен в production
CORS_ORIGINS=https://lana-flowers.vercel.app  # через запятую
```

## Текущее состояние / техдолг

Закрытые приоритеты (см. git history): security hardening (phone leak,
upload MIME-sniff, CORS whitelist, webhook secret); UX (cancel/withdraw
offer, BouquetDetail, BaseSheet, формат-утилиты); realtime через SSE;
thumbnails для каталога; counter через ForceReply в чате.

Открытые техдолги (НЕ критично для текущего масштаба, скорее «когда
вырастем»):
- БД-бэкап не настроен — обязательно перед публичным релизом
- Домен `nip.io` — single point of failure, купить нормальный
- Pagination на `/bouquets/my`, `/offers/sent|received` (взорвётся при росте)
- Rate-limiting на `/offers`, `/upload`
- CI/CD, мониторинг, тесты на транзакции
- usePullToRefresh не v-if-устойчивый (onMounted ловит elRef один раз)
- ContactSheet `tel:` через window.location → должно быть через `tg.openLink`

## Стиль

- Комментарии — на русском, объясняют ПОЧЕМУ (не WHAT)
- Без эмодзи в коде/комментариях кроме сообщений юзеру в UI
- `gofmt` для Go, `vite build` без warning'ов для фронта
- Магические числа выносятся в константы рядом с использованием
- Тяжёлые SQL JOIN'ы кодифицированы в общих хелперах (см. `bouquets/query.go`)
