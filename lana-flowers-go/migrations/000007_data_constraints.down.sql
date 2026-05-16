ALTER TABLE bouquets
  DROP CONSTRAINT IF EXISTS bouquets_category_check,
  DROP CONSTRAINT IF EXISTS bouquets_title_len_check,
  DROP CONSTRAINT IF EXISTS bouquets_desc_len_check,
  DROP CONSTRAINT IF EXISTS bouquets_price_pos_check;

ALTER TABLE offers
  DROP CONSTRAINT IF EXISTS offers_price_pos_check;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_display_name_len_check;
