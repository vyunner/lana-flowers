-- Data integrity constraints — раньше бэк валидировал только частично,
-- БД любой мусор принимала.

-- Категория букета — белый список. Без CHECK можно было создать
-- bouquet с category='zalupa', он в каталоге не показывался (фронт
-- фильтрует), но в БД лежал.
ALTER TABLE bouquets
  ADD CONSTRAINT bouquets_category_check
  CHECK (category IN ('roses', 'peonies', 'wild', 'composition', 'dried', 'all'));

-- Длина title/description — иначе можно отправить 1MB-описание и
-- положить рендер UI.
ALTER TABLE bouquets
  ADD CONSTRAINT bouquets_title_len_check  CHECK (char_length(title) <= 120),
  ADD CONSTRAINT bouquets_desc_len_check   CHECK (char_length(description) <= 2000);

-- Цена должна быть положительной. Сейчас валидируется только в
-- service.go (price <= 0 → error), но напрямую через миграцию/админку
-- можно было записать price=-1.
ALTER TABLE bouquets
  ADD CONSTRAINT bouquets_price_pos_check  CHECK (price > 0);

ALTER TABLE offers
  ADD CONSTRAINT offers_price_pos_check    CHECK (price > 0);

-- Display name — sane upper bound. 64 в коде уже, дублируем в БД.
ALTER TABLE users
  ADD CONSTRAINT users_display_name_len_check CHECK (char_length(display_name) <= 64);
