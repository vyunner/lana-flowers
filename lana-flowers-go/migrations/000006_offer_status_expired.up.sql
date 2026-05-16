-- Добавляем 'expired' в enum offer_status.
-- Используется когда продавец принимает один из нескольких pending-офферов
-- на букет — остальным мы автоматически проставляем статус expired,
-- их buyer'ам летит DM «букет ушёл другому».
--
-- ALTER TYPE ADD VALUE поддерживается в транзакции с PG 12+, на нашей
-- production-БД это ОК. IF NOT EXISTS делает миграцию идемпотентной.
ALTER TYPE offer_status ADD VALUE IF NOT EXISTS 'expired';
