-- ───────────────────────────── ENUMs ─────────────────────────────
CREATE TYPE bouquet_status AS ENUM ('active', 'sold', 'archived');
CREATE TYPE offer_status   AS ENUM ('pending', 'accepted', 'rejected', 'countered', 'cancelled');

-- ───────────────────────────── users ─────────────────────────────
CREATE TABLE users (
    user_id       TEXT PRIMARY KEY,                          -- Telegram user.id как text
    first_name    TEXT        NOT NULL DEFAULT '',
    last_name     TEXT        NOT NULL DEFAULT '',
    username      TEXT        NOT NULL DEFAULT '',
    photo_url     TEXT        NOT NULL DEFAULT '',
    is_premium    BOOLEAN     NOT NULL DEFAULT FALSE,
    language_code TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ───────────────────────────── bouquets ─────────────────────────────
CREATE TABLE bouquets (
    id          BIGSERIAL PRIMARY KEY,
    seller_id   TEXT        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    price       BIGINT      NOT NULL CHECK (price > 0),
    city        TEXT        NOT NULL,
    photos      TEXT[]      NOT NULL DEFAULT '{}',           -- URLs картинок
    category    TEXT        NOT NULL DEFAULT 'all',          -- chip key: roses, peonies, wild, composition, dried
    status      bouquet_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bouquets_active_city     ON bouquets (city, created_at DESC) WHERE status = 'active';
CREATE INDEX idx_bouquets_active_category ON bouquets (category, created_at DESC) WHERE status = 'active';
CREATE INDEX idx_bouquets_seller          ON bouquets (seller_id, created_at DESC);

-- ───────────────────────────── offers ─────────────────────────────
CREATE TABLE offers (
    id           BIGSERIAL PRIMARY KEY,
    bouquet_id   BIGINT      NOT NULL REFERENCES bouquets(id) ON DELETE CASCADE,
    buyer_id     TEXT        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    seller_id    TEXT        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, -- денормализованно для быстрых запросов
    price        BIGINT      NOT NULL CHECK (price > 0),
    message      TEXT        NOT NULL DEFAULT '',
    status       offer_status NOT NULL DEFAULT 'pending',
    parent_id    BIGINT      REFERENCES offers(id) ON DELETE SET NULL,            -- для встречных предложений
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMPTZ
);

CREATE INDEX idx_offers_bouquet         ON offers (bouquet_id, created_at DESC);
CREATE INDEX idx_offers_buyer           ON offers (buyer_id, created_at DESC);
CREATE INDEX idx_offers_seller_pending  ON offers (seller_id) WHERE status = 'pending';
