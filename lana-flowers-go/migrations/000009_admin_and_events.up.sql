-- Админ-панель в отдельном Telegram-боте + persistent лог всех значимых
-- событий маркетплейса.
--
-- admins — кто имеет доступ к админ-боту. notifications jsonb хранит
-- ТОЛЬКО отключённые типы событий: ключи событий с false-значением,
-- отсутствующий ключ = ON по умолчанию. Меньше read'а после сохранения.
--
-- admin_requests — те кто стартанул админ-бот но не админ. Появляются на
-- /start от non-admin, удаляются когда action: либо admin выдал доступ
-- (попали в admins), либо admin отказал (просто удалили).
--
-- event_log — append-only лента всех событий: регистрации, объявления,
-- офферы и их статусы. payload jsonb для гибкости (схема каждого события
-- своя). НЕ отображается нигде в UI пока что, но индексы готовы под
-- будущую админ-страницу истории.

CREATE TABLE admins (
    tg_id text PRIMARY KEY,
    granted_at timestamptz NOT NULL DEFAULT NOW(),
    granted_by text,  -- tg_id админа, выдавшего доступ; NULL для bootstrap-админа
    notifications jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE admin_requests (
    tg_id text PRIMARY KEY,
    display_name text NOT NULL DEFAULT '',
    username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE event_log (
    id BIGSERIAL PRIMARY KEY,
    event_type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- Индекс под "покажи последние N событий" (главный use-case будущей UI).
CREATE INDEX event_log_created_at_idx ON event_log(created_at DESC);
-- Индекс под "покажи последние N событий типа X" (фильтрация в UI).
CREATE INDEX event_log_type_created_idx ON event_log(event_type, created_at DESC);

-- Bootstrap-админ: владелец маркетплейса (vyunner). Без granted_by потому
-- что нет родителя — это первая запись в системе. Дальше доступ выдаётся
-- через UI админ-бота, granted_by заполняется.
INSERT INTO admins (tg_id, granted_at, granted_by) VALUES ('959568457', NOW(), NULL);
