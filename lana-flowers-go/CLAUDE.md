## Lana Flowers — Go API

### Что это
Backend для Telegram Mini App `@lana_flowers_bot` (C2C-маркетплейс букетов с торгом). Stack: Go + Gin + Postgres + golang-migrate. Архитектура зеркалит `mafia-bmb-go`.

### Структура
- `cmd/api/` — entry point + CLI commands (migrate, dbdoc)
- `internal/db/` — Postgres pool
- `internal/auth/` — валидация Telegram WebApp initData (HMAC-SHA256 по bot token)
- `internal/response/` — `OK(c, data)` / `Err(c, status, code, msg)` хелперы
- `internal/handler/<domain>/` — HTTP-обработчики, по доменам (`users`, `bouquets`, `offers`)
- `migrations/` — golang-migrate `.up.sql` / `.down.sql`

### Auth
Все запросы (кроме `/health`) требуют либо:
- `X-Telegram-Init-Data` — initData из Telegram WebApp (валидируется HMAC по bot token), либо
- `X-Secret` — для server-to-server (бот → API, если появится)

Из `init_data` извлекаем `user.id` Telegram → используем как `user_id` в БД (text).

### Миграции
- Локально через SSH-туннель к prod: `./db-migrate.sh up | down [N] | version | force N | reset`
- На сервере: `./app migrate up`
- После любой миграции автоматически перегенерируется `DATABASE.md`

### Деплой
- `./deploy.sh "сообщение"` — git pull/add/commit/push, локально
- `./start-deploy.sh` — на сервере: git pull → go build → systemctl restart

### Локальный dev
1. Скопируй `.env.example` → `.env`, заполни DATABASE_URL и TELEGRAM_BOT_TOKEN
2. `go run ./cmd/api migrate up`
3. `go run ./cmd/api`
4. API на `:8080`. Health: `curl localhost:8080/health`
