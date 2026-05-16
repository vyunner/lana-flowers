-- Один pending-оффер от одного покупателя на один букет. Защита от спама.
CREATE UNIQUE INDEX idx_offers_pending_unique
  ON offers (bouquet_id, buyer_id)
  WHERE status = 'pending';
